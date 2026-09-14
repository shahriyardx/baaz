package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sort"
)

// AvailableHeights reports the video resolutions a media URL actually offers,
// tallest first.
//
// Offering 4K on a 720p video is noise, and worse, it implies a download that
// will silently come back smaller than asked for. yt-dlp already knows the
// answer; this asks it.
func (e *Engine) AvailableHeights(ctx context.Context, url string) ([]int, error) {
	if err := e.ensureMediaTools(ctx, func(string) {}); err != nil {
		return nil, err
	}
	bin, err := e.lookupTool("yt-dlp")
	if err != nil {
		return nil, err
	}

	// -J is one extraction pass and no download. --flat-playlist keeps a
	// playlist URL from expanding into every entry.
	cmd := exec.CommandContext(ctx, bin,
		"-J", "--no-warnings", "--no-playlist", "--skip-download", url)
	cmd.Env = e.ytdlpEnv()
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	// A single video's metadata is comfortably under this; the cap is only
	// so a pathological response cannot exhaust memory.
	data, rerr := io.ReadAll(io.LimitReader(out, 32<<20))
	werr := cmd.Wait()
	if rerr != nil {
		return nil, rerr
	}
	if werr != nil {
		return nil, fmt.Errorf("yt-dlp could not read that link: %w", werr)
	}

	var info struct {
		Formats []struct {
			Height int    `json:"height"`
			VCodec string `json:"vcodec"`
		} `json:"formats"`
	}
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("unexpected yt-dlp output: %w", err)
	}

	seen := map[int]bool{}
	var heights []int
	for _, f := range info.Formats {
		// "none" marks an audio-only stream, which has no resolution to offer.
		if f.Height <= 0 || f.VCodec == "none" {
			continue
		}
		if !seen[f.Height] {
			seen[f.Height] = true
			heights = append(heights, f.Height)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(heights)))
	return heights, nil
}
