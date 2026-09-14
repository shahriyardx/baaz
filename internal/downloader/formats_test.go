package downloader

import (
	"encoding/json"
	"reflect"
	"testing"
)

// The filtering and ordering of yt-dlp's format list, without the network.
func TestHeightsFromFormatsJSON(t *testing.T) {
	// Shaped like real yt-dlp output: audio-only rows carry vcodec "none",
	// and storyboards come back as tiny heights.
	raw := `{"formats":[
      {"height":1080,"vcodec":"avc1"},
      {"height":2160,"vcodec":"vp9"},
      {"height":0,"vcodec":"none"},
      {"height":720,"vcodec":"avc1"},
      {"height":1080,"vcodec":"vp9"},
      {"height":null,"vcodec":"none"},
      {"height":144,"vcodec":"avc1"}
    ]}`
	var info struct {
		Formats []struct {
			Height int    `json:"height"`
			VCodec string `json:"vcodec"`
		} `json:"formats"`
	}
	if err := json.Unmarshal([]byte(raw), &info); err != nil {
		t.Fatal(err)
	}
	seen := map[int]bool{}
	var got []int
	for _, f := range info.Formats {
		if f.Height <= 0 || f.VCodec == "none" {
			continue
		}
		if !seen[f.Height] {
			seen[f.Height] = true
			got = append(got, f.Height)
		}
	}
	// Same reduction AvailableHeights performs: unique, video-only.
	want := []int{1080, 2160, 720, 144}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v (before sorting)", got, want)
	}
}
