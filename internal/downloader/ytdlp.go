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
)

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
func lookupYtdlp() (string, error) {
	if p, err := exec.LookPath("yt-dlp"); err == nil {
		return p, nil
	}
	for _, dir := range ytdlpBinDirs {
		p := filepath.Join(dir, "yt-dlp")
		if st, err := os.Stat(p); err == nil && !st.IsDir() && st.Mode()&0o111 != 0 {
			return p, nil
		}
	}
	return "", fmt.Errorf("yt-dlp is not installed (%s)", ytdlpInstallHint)
}

// ytdlpEnv adds those same directories to the child's PATH, because yt-dlp
// looks up ffmpeg itself and would otherwise fail to merge video with audio
// on exactly the machines described above.
func ytdlpEnv() []string {
	env := os.Environ()
	seen := map[string]bool{}
	var parts []string
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir != "" && !seen[dir] {
			seen[dir] = true
			parts = append(parts, dir)
		}
	}
	for _, dir := range ytdlpBinDirs {
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
	bin, err := lookupYtdlp()
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
	args := []string{"--newline", "--no-playlist", "-c",
		"-P", "home:" + outDir, "-P", "temp:" + tmpDir,
		"-o", "%(title).120B [%(id)s].%(ext)s"}
	args = append(args, formatArgs(j.Format)...)
	args = append(args, j.URL)
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = ytdlpEnv()
	cmd.Stderr = os.Stderr // ends up in the daemon log
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
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
			lastPath = strings.TrimSpace(m[1])
			j.mu.Lock()
			j.Filename = filepath.Base(lastPath)
			j.mu.Unlock()
		}
		if m := ytExtractRe.FindStringSubmatch(line); m != nil {
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
		if ctx.Err() != nil {
			return ctx.Err() // paused/canceled; `-c` resumes on next run
		}
		return fmt.Errorf("yt-dlp failed: %v (see daemon.log)", err)
	}
	j.mu.Lock()
	if j.Total > 0 {
		atomic.StoreInt64(&seg.Written, j.Total)
	}
	if lastPath != "" {
		// Never leave FinalPath pointing into the hidden temp dir: if the
		// announced path lives there, the finished file is its basename in
		// the real output dir.
		if strings.HasPrefix(lastPath, tmpDir) {
			if moved := filepath.Join(outDir, filepath.Base(lastPath)); fileExists(moved) {
				lastPath = moved
			}
		}
		j.FinalPath = lastPath
	}
	j.mu.Unlock()
	return nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// formatArgs maps a quality preset to yt-dlp flags. Quality uses -S res
// sorting rather than a height filter: "res" is the smaller dimension, so
// vertical videos (reels: 720x1280) work, and sorting always picks the
// closest available format instead of erroring like a strict -f filter.
func formatArgs(preset string) []string {
	switch preset {
	case "1080", "720", "480":
		return []string{"-S", "res:" + preset}
	case "audio":
		return []string{"-f", "bestaudio/b", "-x", "--audio-format", "mp3"}
	default: // "", "best"
		return nil
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
