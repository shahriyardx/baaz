package downloader

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// Reads a real video's resolutions, because the whole point is that the list
// reflects what a given video actually has rather than a fixed menu.
func TestAvailableHeightsReadsRealVideos(t *testing.T) {
	if testing.Short() {
		t.Skip("network test")
	}
	e := NewEngine(8, 1<<20)
	e.Client = &http.Client{Timeout: 90 * time.Second}

	// Asserting an exact maximum bakes in what a video offered on the day
	// the test was written — one of these was remastered to 4K and broke it.
	// What must hold is the shape: real video heights, tallest first.
	for _, url := range []string{
		"https://www.youtube.com/watch?v=aqz-KE-bpKQ",
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
	} {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		heights, err := e.AvailableHeights(ctx, url)
		cancel()
		if err != nil {
			t.Skipf("%v", err) // offline, or yt-dlp unavailable
		}
		if len(heights) == 0 {
			t.Errorf("%s: no resolutions found", url)
			continue
		}
		for i, h := range heights {
			// Storyboard entries come back as tiny heights; they are not
			// something anyone can choose to download.
			if h < 100 || h > 8000 {
				t.Errorf("%s: implausible height %d in %v", url, h, heights)
			}
			if i > 0 && h >= heights[i-1] {
				t.Errorf("%s: not sorted tallest-first: %v", url, heights)
				break
			}
		}
		t.Logf("%s -> %v", url[:40], heights)
	}
}
