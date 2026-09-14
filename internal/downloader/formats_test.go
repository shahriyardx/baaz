package downloader

import (
	"encoding/json"
	"sort"
	"testing"
)

// reduce mirrors what AvailableQualities does with yt-dlp's format list, so
// the filtering and labelling can be tested without the network.
func reduce(t *testing.T, raw string) []Quality {
	t.Helper()
	var info struct {
		Formats []struct {
			Height int    `json:"height"`
			Width  int    `json:"width"`
			VCodec string `json:"vcodec"`
		} `json:"formats"`
	}
	if err := json.Unmarshal([]byte(raw), &info); err != nil {
		t.Fatal(err)
	}
	byHeight := map[int]Quality{}
	for _, f := range info.Formats {
		family := codecFamily(f.VCodec)
		if f.Height <= 0 || family == "" {
			continue
		}
		label := f.Height
		if f.Width > 0 && f.Width < f.Height {
			label = f.Width
		}
		q := Quality{Height: f.Height, Width: f.Width, Label: label, Codec: family}
		if existing, ok := byHeight[f.Height]; ok && existing.Codec == "h264" {
			continue
		}
		byHeight[f.Height] = q
	}
	out := make([]Quality, 0, len(byHeight))
	for _, q := range byHeight {
		out = append(out, q)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Height > out[b].Height })
	return out
}

// A portrait video is 1080p, not 1920p. Labelling by height alone put
// "1920p" and "1280p" in the menu.
func TestPortraitVideoIsLabelledByItsShortSide(t *testing.T) {
	got := reduce(t, `{"formats":[
      {"width":1080,"height":1920,"vcodec":"avc1.4d402a"},
      {"width":720,"height":1280,"vcodec":"avc1.4d401f"}
    ]}`)
	want := []int{1080, 720}
	if len(got) != len(want) {
		t.Fatalf("got %d qualities, want %d: %+v", len(got), len(want), got)
	}
	for i, q := range got {
		if q.Label != want[i] {
			t.Errorf("height %d labelled %dp, want %dp", q.Height, q.Label, want[i])
		}
	}
}

// Landscape keeps using the height, which is the number people quote.
func TestLandscapeVideoIsLabelledByHeight(t *testing.T) {
	got := reduce(t, `{"formats":[{"width":1920,"height":1080,"vcodec":"avc1"}]}`)
	if len(got) != 1 || got[0].Label != 1080 {
		t.Errorf("got %+v, want a single 1080p", got)
	}
}

// When a resolution offers several codecs, H.264 wins: it is the one that
// plays everywhere, so the menu should not mark it as VP9.
func TestH264WinsWhenAResolutionOffersSeveralCodecs(t *testing.T) {
	for _, order := range []string{
		`{"formats":[{"height":1080,"width":1920,"vcodec":"vp9"},{"height":1080,"width":1920,"vcodec":"avc1"}]}`,
		`{"formats":[{"height":1080,"width":1920,"vcodec":"avc1"},{"height":1080,"width":1920,"vcodec":"vp9"}]}`,
	} {
		got := reduce(t, order)
		if len(got) != 1 {
			t.Fatalf("expected one 1080p entry, got %+v", got)
		}
		if got[0].Codec != "h264" {
			t.Errorf("codec %q won over h264 (input order matters, it should not)", got[0].Codec)
		}
	}
}

// Audio-only rows carry vcodec "none" and have no resolution to offer.
func TestAudioOnlyRowsAreDropped(t *testing.T) {
	got := reduce(t, `{"formats":[
      {"height":0,"vcodec":"none"},
      {"height":720,"width":1280,"vcodec":"avc1"}
    ]}`)
	if len(got) != 1 || got[0].Label != 720 {
		t.Errorf("got %+v, want only 720p", got)
	}
}

func TestCodecFamilies(t *testing.T) {
	cases := map[string]string{
		"avc1.64002a": "h264", "h264": "h264",
		"vp9": "vp9", "vp09.00.50.08": "vp9",
		"av01.0.09M.08": "av1", "av1": "av1",
		"none": "", "": "",
		"theora": "other",
	}
	for in, want := range cases {
		if got := codecFamily(in); got != want {
			t.Errorf("codecFamily(%q) = %q, want %q", in, got, want)
		}
	}
}
