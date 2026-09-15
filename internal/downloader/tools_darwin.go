package downloader

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Media downloads need yt-dlp, and yt-dlp needs ffmpeg: YouTube serves video
// and audio as separate streams, so without ffmpeg there is nothing to merge
// and even "best" fails. macOS has no package manager to lean on, so baaz
// fetches both itself the first time they are needed and keeps them in its
// own directory. Nothing is redistributed — each comes from its upstream
// release — and the download is checked against the SHA-256 that release
// publishes.
const (
	ytdlpRepo  = "yt-dlp/yt-dlp"
	ytdlpAsset = "yt-dlp_macos" // universal binary
	ffmpegRepo = "eugeneware/ffmpeg-static"
)

func ffmpegAsset() string {
	if runtime.GOARCH == "arm64" {
		return "ffmpeg-darwin-arm64.gz"
	}
	return "ffmpeg-darwin-x64.gz"
}

// ensureMediaTools provisions anything missing. Already-installed copies —
// Homebrew's, MacPorts', or a previous fetch — are used as they are.
func (e *Engine) ensureMediaTools(ctx context.Context, note func(string)) error {
	if e.ToolsDir == "" {
		return nil // provisioning disabled; fall back to whatever is on PATH
	}
	if _, err := e.lookupTool("yt-dlp"); err != nil {
		note("getting yt-dlp (one time)")
		if err := e.fetchTool(ctx, ytdlpRepo, ytdlpAsset, "yt-dlp", false); err != nil {
			return fmt.Errorf("could not download the video tools — check your connection: %w", err)
		}
	}
	if _, err := e.lookupTool("ffmpeg"); err != nil {
		note("getting ffmpeg (one time)")
		if err := e.fetchTool(ctx, ffmpegRepo, ffmpegAsset(), "ffmpeg", true); err != nil {
			return fmt.Errorf("could not download the video tools — check your connection: %w", err)
		}
	}
	return nil
}

type ghAsset struct {
	Name   string `json:"name"`
	URL    string `json:"browser_download_url"`
	Digest string `json:"digest"` // "sha256:<hex>"
}

// fetchTool downloads one asset from a repo's latest release, verifies it
// against the digest the API reports, and installs it into ToolsDir.
func (e *Engine) fetchTool(ctx context.Context, repo, asset, name string, gz bool) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://api.github.com/repos/"+repo+"/releases/latest", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := e.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("release lookup: HTTP %s", resp.Status)
	}
	var rel struct {
		Assets []ghAsset `json:"assets"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&rel); err != nil {
		return err
	}
	var found *ghAsset
	for i := range rel.Assets {
		if rel.Assets[i].Name == asset {
			found = &rel.Assets[i]
			break
		}
	}
	if found == nil {
		return fmt.Errorf("%s has no asset %q", repo, asset)
	}

	if err := os.MkdirAll(e.ToolsDir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(e.ToolsDir, "."+name+"-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	dreq, err := http.NewRequestWithContext(ctx, http.MethodGet, found.URL, nil)
	if err != nil {
		tmp.Close()
		return err
	}
	dresp, err := e.Client.Do(dreq)
	if err != nil {
		tmp.Close()
		return err
	}
	defer dresp.Body.Close()
	if dresp.StatusCode != http.StatusOK {
		tmp.Close()
		return fmt.Errorf("download: HTTP %s", dresp.Status)
	}

	// Hash the bytes as delivered, since that is what the digest covers;
	// decompression happens on the way to disk.
	sum := sha256.New()
	body := io.TeeReader(dresp.Body, sum)
	var src io.Reader = body
	if gz {
		zr, err := gzip.NewReader(body)
		if err != nil {
			tmp.Close()
			return err
		}
		defer zr.Close()
		src = zr
	}
	if _, err := io.Copy(tmp, src); err != nil {
		tmp.Close()
		return err
	}
	if gz {
		// Drain whatever gzip left, or the hash covers only part of the file.
		io.Copy(io.Discard, body)
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if want, ok := strings.CutPrefix(found.Digest, "sha256:"); ok && want != "" {
		got := hex.EncodeToString(sum.Sum(nil))
		if got != want {
			return fmt.Errorf("%s failed its checksum (got %s, want %s)", asset, got[:12], want[:12])
		}
	}

	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(e.ToolsDir, name))
}
