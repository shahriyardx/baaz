package downloader

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"
)

// Quality is one resolution a media URL offers.
type Quality struct {
	// Height and Width as reported. Label is the number people recognise:
	// the smaller side, so a 1080x1920 portrait video reads 1080p rather
	// than 1920p.
	Height int    `json:"height"`
	Width  int    `json:"width"`
	Label  int    `json:"label"`
	Codec  string `json:"codec"` // h264, vp9, av1, or other
}

// codecFamily reduces yt-dlp's precise codec strings to what matters: H.264
// plays in everything, the rest may not.
func codecFamily(vcodec string) string {
	switch {
	case vcodec == "" || vcodec == "none":
		return ""
	case strings.HasPrefix(vcodec, "avc") || strings.HasPrefix(vcodec, "h264"):
		return "h264"
	case strings.HasPrefix(vcodec, "vp9") || strings.HasPrefix(vcodec, "vp09"):
		return "vp9"
	case strings.HasPrefix(vcodec, "av01") || strings.HasPrefix(vcodec, "av1"):
		return "av1"
	default:
		return "other"
	}
}

// AvailableQualities reports the resolutions a media URL actually offers,
// largest first, with the best codec available at each.
//
// Offering 4K on a 720p video is noise, and worse, implies a download that
// will silently come back smaller than asked for. yt-dlp already knows the
// answer; this asks it. The codec comes along because whether a file plays
// is a property of the codec, not the resolution — saying "needs VLC" for
// everything above 1080p is a guess that happens to fit YouTube and not
// much else.
// errToolsNotReady means the quality list cannot be offered yet because the
// video tools have not been fetched. The first download will fetch them.
var errToolsNotReady = errors.New("video tools are not ready yet")

func (e *Engine) AvailableQualities(ctx context.Context, url string) ([]Quality, error) {
	// Deliberately does not fetch the video tools. This runs when a menu is
	// opening, with nowhere to report progress — on a machine that has none
	// of them yet it downloaded around 80MB in silence while the menu read
	// "Checking qualities…", and on a slow line it simply timed out.
	//
	// Missing tools are an answer, not an error to solve here: the menu falls
	// back to Best and Audio, both of which work, and the download that
	// follows does the fetching where there is a row to show progress on.
	bin, err := e.lookupTool("yt-dlp")
	if err != nil {
		return nil, errToolsNotReady
	}

	// -J is one extraction pass and no download. --no-playlist keeps a
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
		return nil, fmt.Errorf("could not read that link")
	}

	var info struct {
		Formats []struct {
			Height int    `json:"height"`
			Width  int    `json:"width"`
			VCodec string `json:"vcodec"`
		} `json:"formats"`
	}
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("could not read that link")
	}

	byHeight := map[int]Quality{}
	for _, f := range info.Formats {
		family := codecFamily(f.VCodec)
		// An audio-only stream has no resolution to offer.
		if f.Height <= 0 || family == "" {
			continue
		}
		label := f.Height
		if f.Width > 0 && f.Width < f.Height {
			label = f.Width // portrait: the smaller side is the one quoted
		}
		q := Quality{Height: f.Height, Width: f.Width, Label: label, Codec: family}
		// Keep H.264 when the same resolution offers several codecs, since
		// that is the one that plays everywhere.
		if existing, ok := byHeight[f.Height]; ok && existing.Codec == "h264" {
			continue
		}
		byHeight[f.Height] = q
	}

	qualities := make([]Quality, 0, len(byHeight))
	for _, q := range byHeight {
		qualities = append(qualities, q)
	}
	sort.Slice(qualities, func(a, b int) bool { return qualities[a].Height > qualities[b].Height })
	return qualities, nil
}
