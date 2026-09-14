package downloader

import (
	"strings"
	"testing"
)

func argsFor(preset string) string { return strings.Join(formatArgs(preset), " ") }

// A .webm that QuickTime refuses is not a successful download. Every video
// preset has to ask for H.264/AAC in an mp4 container.
func TestVideoPresetsAskForAPlayableFile(t *testing.T) {
	for _, preset := range []string{"", "best", "2160", "1440", "1080", "720", "480"} {
		got := argsFor(preset)
		for _, want := range []string{"vcodec:h264", "acodec:aac", "--merge-output-format mp4"} {
			if !strings.Contains(got, want) {
				t.Errorf("preset %q is missing %q:\n  %s", preset, want, got)
			}
		}
	}
}

// The resolution presets have to actually request that resolution — the
// point of 4K is getting 4K, even though it costs H.264.
func TestResolutionPresetsPassTheirHeight(t *testing.T) {
	for _, preset := range []string{"2160", "1440", "1080", "720", "480"} {
		if got := argsFor(preset); !strings.Contains(got, "res:"+preset) {
			t.Errorf("preset %q does not request res:%s: %s", preset, preset, got)
		}
	}
	// "best" must not pin a resolution, or it would not be best.
	if got := argsFor("best"); strings.Contains(got, "res:") {
		t.Errorf(`"best" should not cap resolution: %s`, got)
	}
}

// Audio-only is extracted to mp3 and must not drag in video sorting.
func TestAudioPresetExtractsMP3(t *testing.T) {
	got := argsFor("audio")
	for _, want := range []string{"-x", "--audio-format mp3", "bestaudio"} {
		if !strings.Contains(got, want) {
			t.Errorf("audio preset missing %q: %s", want, got)
		}
	}
	if strings.Contains(got, "merge-output-format") {
		t.Errorf("audio preset should not merge into a video container: %s", got)
	}
}

// An unknown preset must behave like "best" rather than silently passing
// nothing, which is what produced .webm files in the first place.
func TestUnknownPresetFallsBackToPlayable(t *testing.T) {
	got := argsFor("nonsense")
	if !strings.Contains(got, "vcodec:h264") {
		t.Errorf("unknown preset should still ask for a playable file: %q", got)
	}
}
