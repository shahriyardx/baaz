package downloader

import (
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Media downloads need yt-dlp, and yt-dlp needs ffmpeg: YouTube serves video
// and audio as separate streams, so without ffmpeg there is nothing to merge
// and even "best" fails. macOS has no package manager to lean on, so baaz
// fetches both itself the first time they are needed and keeps them in its
// own directory. Nothing is redistributed — each comes from its upstream
// release.
//
// Everything here goes through github.com/<repo>/releases/latest/download/…,
// never api.github.com. The API allows 60 unauthenticated calls an hour *per
// IP address*, which sounds generous until you are behind carrier-grade NAT
// sharing one address with a whole neighbourhood: the budget is long gone
// before you arrive, and the install fails with a bare 403 through no fault
// of yours. The plain download path carries no such limit.
const (
	ytdlpRepo  = "yt-dlp/yt-dlp"
	ytdlpAsset = "yt-dlp_macos" // universal binary
	ytdlpSums  = "SHA2-256SUMS"
	ffmpegRepo = "eugeneware/ffmpeg-static"
)

func ffmpegAsset() string {
	if runtime.GOARCH == "arm64" {
		return "ffmpeg-darwin-arm64.gz"
	}
	return "ffmpeg-darwin-x64.gz"
}

func releaseURL(repo, asset string) string {
	return "https://github.com/" + repo + "/releases/latest/download/" + asset
}

// ensureMediaTools provisions anything missing. Already-installed copies —
// Homebrew's, MacPorts', or a previous fetch — are used as they are.
func (e *Engine) ensureMediaTools(ctx context.Context, note func(string)) error {
	if e.ToolsDir == "" {
		return nil // provisioning disabled; fall back to whatever is on PATH
	}
	if _, err := e.lookupTool("yt-dlp"); err != nil {
		note("getting yt-dlp (one time)")
		// yt-dlp publishes SHA2-256SUMS alongside the binary, so this one
		// can be checked against the project's own figure.
		if err := e.fetchTool(ctx, fetchSpec{
			url:     releaseURL(ytdlpRepo, ytdlpAsset),
			sumsURL: releaseURL(ytdlpRepo, ytdlpSums),
			sumName: ytdlpAsset,
			name:    "yt-dlp",
			verify:  []string{"--version"},
		}); err != nil {
			return err
		}
	}
	if _, err := e.lookupTool("ffmpeg"); err != nil {
		note("getting ffmpeg (one time)")
		// ffmpeg-static publishes no checksum file. The bytes still arrive
		// over TLS from the project's own release, and the result has to
		// unpack and actually run before it is installed.
		if err := e.fetchTool(ctx, fetchSpec{
			url:    releaseURL(ffmpegRepo, ffmpegAsset()),
			gz:     true,
			name:   "ffmpeg",
			verify: []string{"-version"},
		}); err != nil {
			return err
		}
	}
	return nil
}

type fetchSpec struct {
	url     string   // direct release download
	sumsURL string   // optional: a SHA256SUMS-style file to check against
	sumName string   // the entry to look for in that file
	name    string   // what to install it as
	gz      bool     // the asset is gzipped
	verify  []string // args that must exit 0 once installed
}

// fetchTool downloads one tool, checks it, and installs it into ToolsDir.
func (e *Engine) fetchTool(ctx context.Context, spec fetchSpec) error {
	want := ""
	if spec.sumsURL != "" {
		var err error
		if want, err = e.expectedSum(ctx, spec.sumsURL, spec.sumName); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(e.ToolsDir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(e.ToolsDir, "."+spec.name+"-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	body, closeBody, err := e.get(ctx, spec.url)
	if err != nil {
		tmp.Close()
		return err
	}
	defer closeBody()

	// Hash the bytes as delivered, since that is what a published sum
	// covers; decompression happens on the way to disk.
	sum := sha256.New()
	tee := io.TeeReader(body, sum)
	var src io.Reader = tee
	if spec.gz {
		zr, err := gzip.NewReader(tee)
		if err != nil {
			tmp.Close()
			return fmt.Errorf("the download was not readable")
		}
		defer zr.Close()
		src = zr
	}
	if _, err := io.Copy(tmp, src); err != nil {
		tmp.Close()
		return err
	}
	if spec.gz {
		// Drain whatever gzip left, or the hash covers only part of it.
		io.Copy(io.Discard, tee)
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if want != "" {
		if got := hex.EncodeToString(sum.Sum(nil)); got != want {
			return fmt.Errorf("%s failed its checksum (got %s, expected %s)",
				spec.name, got[:12], want[:12])
		}
	}
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		return err
	}
	// Run it before trusting it. Without a published checksum this is the
	// only thing standing between a truncated or wrong-architecture download
	// and a video download that fails later for no visible reason.
	if len(spec.verify) > 0 {
		if err := runsOK(ctx, tmp.Name(), spec.verify); err != nil {
			return fmt.Errorf("the downloaded %s did not run: %w", spec.name, err)
		}
	}
	return os.Rename(tmp.Name(), filepath.Join(e.ToolsDir, spec.name))
}

// expectedSum pulls one entry out of a SHA256SUMS-style file.
func (e *Engine) expectedSum(ctx context.Context, url, name string) (string, error) {
	body, closeBody, err := e.get(ctx, url)
	if err != nil {
		return "", err
	}
	defer closeBody()
	sc := bufio.NewScanner(io.LimitReader(body, 1<<20))
	for sc.Scan() {
		// "<hex>  <name>", with one or two spaces and an optional "*".
		fields := strings.Fields(sc.Text())
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == name {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("no checksum published for %s", name)
}

// get performs a GET and turns a failure into something worth reading. A 403
// used to arrive as a bare "HTTP 403 Forbidden" with no hint of what to do.
func (e *Engine) get(ctx context.Context, url string) (io.Reader, func(), error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, func() {}, err
	}
	resp, err := e.Client.Do(req)
	if err != nil {
		return nil, func() {}, fmt.Errorf("could not reach github.com — check your connection")
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		switch resp.StatusCode {
		case http.StatusForbidden, http.StatusTooManyRequests:
			// Kept short: the download list shows two lines of it.
			return nil, func() {}, fmt.Errorf(
				"GitHub refused the download. Try again shortly, or run: brew install yt-dlp ffmpeg")
		case http.StatusNotFound:
			return nil, func() {}, fmt.Errorf("that download is no longer published (%s)", resp.Status)
		default:
			return nil, func() {}, fmt.Errorf("github.com returned %s", resp.Status)
		}
	}
	return resp.Body, func() { resp.Body.Close() }, nil
}

func runsOK(ctx context.Context, bin string, args []string) error {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}
