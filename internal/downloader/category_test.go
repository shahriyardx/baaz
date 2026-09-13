package downloader

import (
	"path/filepath"
	"testing"
)

func TestCategoryFor(t *testing.T) {
	cases := map[string]string{
		"clip.mp4": "Videos", "song.MP3": "Music", "paper.pdf": "Documents",
		"tool.dmg": "Other", "archive.tar.gz": "Compressed", "pic.jpeg": "Images",
		"installer.deb": "Programs", "noext": "Other", "": "Other",
	}
	for name, want := range cases {
		if got := CategoryFor(name); got != want {
			t.Errorf("CategoryFor(%q) = %q, want %q", name, got, want)
		}
	}
}

// Files land directly in the download folder under their category — there is
// no baaz-branded folder in between.
func TestCategorizedDirHasNoWrapperFolder(t *testing.T) {
	base := filepath.Join("/home", "u", "Downloads")
	got := CategorizedDir(base, "clip.mp4")
	want := filepath.Join(base, "Videos")
	if got != want {
		t.Fatalf("CategorizedDir = %q, want %q", got, want)
	}
	if filepath.Base(filepath.Dir(got)) != "Downloads" {
		t.Errorf("category folder should sit directly in the download dir, got %q", got)
	}
}
