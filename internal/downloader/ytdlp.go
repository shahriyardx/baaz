package downloader

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// rateSettle is how long the speed limit must hold steady before a running
// yt-dlp is restarted to pick it up.
const rateSettle = 600 * time.Millisecond

const (
	KindHTTP  = ""      // plain segmented/stream download (zero value: older state files)
	KindMedia = "media" // handled by yt-dlp
)

// Hosts whose page URLs carry media that only yt-dlp can extract.
var mediaHosts = []string{
	"youtube.com", "youtu.be", "vimeo.com", "twitch.tv", "tiktok.com",
	"x.com", "twitter.com", "instagram.com", "facebook.com",
	"dailymotion.com", "soundcloud.com",
}

func IsMediaURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	h := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	for _, m := range mediaHosts {
		if h == m || strings.HasSuffix(h, "."+m) {
			return true
		}
	}
	return false
}

var (
	ytProgressRe = regexp.MustCompile(`\[download\]\s+([\d.]+)% of ~?\s*([\d.]+)([KMGT])iB`)
	ytDestRe     = regexp.MustCompile(`\[download\] Destination: (.+)$`)
	ytMergeRe    = regexp.MustCompile(`\[Merger\] Merging formats into "(.+)"`)
	ytExtractRe  = regexp.MustCompile(`\[ExtractAudio\] Destination: (.+)$`)
	ytAlreadyRe  = regexp.MustCompile(`\[download\] (.+) has already been downloaded`)
	// temp-path workflow: the finished file's real home is announced here
	ytMoveRe = regexp.MustCompile(`\[MoveFiles\] Moving file "(.+)" to "(.+)"`)
)

// lookupYtdlp resolves the yt-dlp binary.
//
// PATH alone is not enough. The daemon is usually started by something with a
// minimal environment — Chrome launching the native-messaging host, or a GUI
// login item — and that PATH omits the directories package managers install
// into (notably Homebrew's /opt/homebrew/bin on Apple Silicon). Falling back
// to the known locations is what stops "yt-dlp is not installed" from being
// reported on a machine where it plainly is.
func (e *Engine) lookupTool(name string) (string, error) {
	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}
	for _, dir := range e.toolDirs() {
		p := filepath.Join(dir, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() && st.Mode()&0o111 != 0 {
			return p, nil
		}
	}
	return "", fmt.Errorf("%s is not installed (%s)", name, ytdlpInstallHint)
}

// toolDirs is where to look besides PATH: baaz's own directory first, then
// the package managers'.
func (e *Engine) toolDirs() []string {
	if e.ToolsDir == "" {
		return ytdlpBinDirs
	}
	return append([]string{e.ToolsDir}, ytdlpBinDirs...)
}

// ytdlpEnv adds those same directories to the child's PATH, because yt-dlp
// looks up ffmpeg itself and would otherwise fail to merge video with audio
// on exactly the machines described above.
func (e *Engine) ytdlpEnv() []string {
	env := os.Environ()
	seen := map[string]bool{}
	var parts []string
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir != "" && !seen[dir] {
			seen[dir] = true
			parts = append(parts, dir)
		}
	}
	for _, dir := range e.toolDirs() {
		if !seen[dir] {
			seen[dir] = true
			parts = append(parts, dir)
		}
	}
	path := "PATH=" + strings.Join(parts, string(filepath.ListSeparator))
	for i, kv := range env {
		if strings.HasPrefix(kv, "PATH=") {
			env[i] = path
			return env
		}
	}
	return append(env, path)
}

// runYtdlp delegates a media-page URL to yt-dlp, translating its progress
// lines into the job's normal accounting. `-c` makes kill-and-rerun resume.
func (e *Engine) runYtdlp(ctx context.Context, j *Job) error {
	// Fetch yt-dlp and ffmpeg if this platform provisions them and they are
	// missing. Clearing the note matters: it is shown to the user.
	if err := e.ensureMediaTools(ctx, j.SetNote); err != nil {
		j.SetNote("")
		return err
	}
	j.SetNote("")

	bin, err := e.lookupTool("yt-dlp")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(j.Dir, 0o755); err != nil {
		return err
	}
	j.mu.Lock()
	if len(j.Segments) == 0 {
		j.Segments = []*Segment{{Start: 0, End: -1}}
	}
	seg := j.Segments[0]
	j.mu.Unlock()

	// .120B truncates the title to 120 bytes: Facebook uses whole captions
	// as titles, which blow past the 255-byte filename limit.
	outDir := j.Dir
	if j.Categorize {
		cat := "Videos"
		if j.Format == "audio" {
			cat = "Music"
		}
		outDir = filepath.Join(j.Dir, cat)
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return err
		}
	}
	tmpDir := filepath.Join(j.Dir, ".baaz-tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return err
	}
	// temp path keeps .part/.ytdl clutter out of the visible folder; the
	// finished file lands in the home path.
	baseArgs := []string{"--newline", "--no-playlist", "-c",
		"-P", "home:" + outDir, "-P", "temp:" + tmpDir,
		"-o", "%(title).120B [%(id)s].%(ext)s"}
	baseArgs = append(baseArgs, formatArgs(j.Format)...)
	baseArgs = append(baseArgs, j.URL)

	// yt-dlp does its own transfers, so the token bucket never sees them;
	// hand it the same cap instead. It reads --limit-rate once at startup,
	// though, so a cap changed mid-download cannot be pushed into the
	// running child the way plain HTTP picks it up on the next chunk. Rerun
	// it instead: `-c` resumes from the .part file, which is exactly what a
	// manual pause/resume did, only without the user having to do it.
	var lastPath string
	for {
		// Subscribe before reading the rate. The other order has a window
		// where a change lands in between and is then waited on through the
		// already-replaced channel, so it would never arrive.
		rateChanged := e.Limiter.Changed()
		rate := e.Limiter.Rate()
		args := baseArgs
		if kb := rate >> 10; kb > 0 {
			args = append([]string{"--limit-rate", fmt.Sprintf("%dK", kb)}, baseArgs...)
		}
		path, rerun, err := e.ytdlpAttempt(ctx, j, seg, bin, args, rate, rateChanged)
		if path != "" {
			lastPath = path
		}
		if rerun {
			continue
		}
		if err != nil {
			return err
		}
		break
	}

	if lastPath != "" && strings.HasPrefix(lastPath, tmpDir) {
		// Never leave FinalPath pointing into the hidden temp dir: if the
		// announced path lives there, the finished file is its basename in
		// the real output dir.
		if moved := filepath.Join(outDir, filepath.Base(lastPath)); fileExists(moved) {
			lastPath = moved
		}
	}

	// Take the size from the finished file rather than from the progress
	// lines. yt-dlp prints no progress at all when the file was already
	// downloaded, which left the job showing "0B" next to a green tick; and
	// where it does print, the figure is the largest single stream, not the
	// merged result.
	size := j.Total
	if lastPath != "" {
		if st, err := os.Stat(lastPath); err == nil && st.Size() > 0 {
			size = st.Size()
		}
	}

	j.mu.Lock()
	if size > 0 {
		j.Total = size
		atomic.StoreInt64(&seg.Written, size)
	}
	if lastPath != "" {
		j.FinalPath = lastPath
	}
	j.mu.Unlock()
	return nil
}

// ytdlpAttempt runs yt-dlp once. It reports the last path yt-dlp announced,
// and whether it was stopped because the speed limit changed — in which case
// the caller runs it again with the new cap rather than treating the
// cancellation as a pause.
func (e *Engine) ytdlpAttempt(ctx context.Context, j *Job, seg *Segment, bin string, args []string, launchRate int64, rateChanged <-chan struct{}) (string, bool, error) {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Once the transfer is done and ffmpeg is merging, restarting would only
	// discard that work, and the cap has nothing left to apply to.
	var merging atomic.Bool
	var rerun atomic.Bool
	watchDone := make(chan struct{})
	defer close(watchDone)

	go func() {
		select {
		case <-rateChanged:
		case <-watchDone:
			return
		}
		// A stepper held down emits a change per click. Wait for the value
		// to settle so a run of clicks costs one restart, not eight.
		for {
			select {
			case <-e.Limiter.Changed():
			case <-watchDone:
				return
			case <-time.After(rateSettle):
				if e.Limiter.Rate() == launchRate || merging.Load() {
					return // back where it started, or too late to matter
				}
				rerun.Store(true)
				cancel()
				return
			}
		}
	}()

	cmd := exec.CommandContext(runCtx, bin, args...)
	cmd.Env = e.ytdlpEnv()
	cmd.Stderr = os.Stderr // ends up in the daemon log
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", false, err
	}
	if err := cmd.Start(); err != nil {
		return "", false, err
	}

	var lastPath string
	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		line := sc.Text()
		if m := ytProgressRe.FindStringSubmatch(line); m != nil {
			pct, _ := strconv.ParseFloat(m[1], 64)
			size, _ := strconv.ParseFloat(m[2], 64)
			total := int64(size * float64(sizeUnit(m[3])))
			// Multi-phase downloads (video then audio) restart the percent;
			// keep the largest total and never let progress move backward.
			j.mu.Lock()
			if total > j.Total {
				j.Total = total
			}
			j.mu.Unlock()
			done := int64(pct / 100 * float64(total))
			if done > atomic.LoadInt64(&seg.Written) {
				atomic.StoreInt64(&seg.Written, done)
			}
			if e.OnProgress != nil {
				e.OnProgress(j)
			}
			continue
		}
		if m := ytDestRe.FindStringSubmatch(line); m != nil {
			lastPath = strings.TrimSpace(m[1])
			j.mu.Lock()
			j.Filename = filepath.Base(lastPath)
			j.mu.Unlock()
		}
		if m := ytMergeRe.FindStringSubmatch(line); m != nil {
			merging.Store(true)
			lastPath = strings.TrimSpace(m[1])
			j.mu.Lock()
			j.Filename = filepath.Base(lastPath)
			j.mu.Unlock()
		}
		if m := ytExtractRe.FindStringSubmatch(line); m != nil {
			merging.Store(true)
			lastPath = strings.TrimSpace(m[1])
			j.mu.Lock()
			j.Filename = filepath.Base(lastPath)
			j.mu.Unlock()
		}
		if m := ytAlreadyRe.FindStringSubmatch(line); m != nil {
			lastPath = strings.TrimSpace(m[1])
			j.mu.Lock()
			j.Filename = filepath.Base(lastPath)
			j.mu.Unlock()
		}
		if m := ytMoveRe.FindStringSubmatch(line); m != nil {
			lastPath = strings.TrimSpace(m[2])
			j.mu.Lock()
			j.Filename = filepath.Base(lastPath)
			j.mu.Unlock()
		}
	}

	if err := cmd.Wait(); err != nil {
		if rerun.Load() {
			return lastPath, true, nil // our own cancel: rerun with the new cap
		}
		if ctx.Err() != nil {
			return lastPath, false, ctx.Err() // paused/canceled; `-c` resumes
		}
		return lastPath, false, fmt.Errorf("could not save that video")
	}
	return lastPath, false, nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// formatArgs maps a quality preset to yt-dlp flags. Quality uses -S res
// sorting rather than a height filter: "res" is the smaller dimension, so
// vertical videos (reels: 720x1280) work, and sorting always picks the
// closest available format instead of erroring like a strict -f filter.
// formatArgs turns a quality preset into yt-dlp arguments.
//
// Left to itself, yt-dlp takes the highest-quality streams, which on YouTube
// means VP9 or AV1 in a .webm container — a file QuickTime, Preview, iMovie
// and iOS all refuse. So the default asks for H.264 with AAC, the
// combination every player handles, merged into an mp4.
//
// That caps the default at 1080p, because YouTube publishes no H.264 above
// it. Rather than quietly hand back a 4K file that will not open, the higher
// resolutions are their own presets: choosing 1440p or 2160p is choosing
// VP9 or AV1, and the menu says so.
func formatArgs(preset string) []string {
	// Sorted after resolution, so it only decides between streams of the
	// same size — a preference, never a cap on what the user asked for.
	const playable = "vcodec:h264,acodec:aac"

	switch preset {
	case "2160", "1440", "1080", "720", "480":
		return []string{
			"-S", "res:" + preset + "," + playable,
			"--merge-output-format", "mp4",
		}
	case "audio":
		return []string{"-f", "bestaudio/b", "-x", "--audio-format", "mp3"}
	default: // "", "best"
		return []string{
			"-S", playable,
			"--merge-output-format", "mp4",
		}
	}
}

func sizeUnit(s string) int64 {
	switch s {
	case "K":
		return 1 << 10
	case "M":
		return 1 << 20
	case "G":
		return 1 << 30
	case "T":
		return 1 << 40
	}
	return 1
}
