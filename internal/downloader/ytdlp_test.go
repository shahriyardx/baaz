package downloader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The daemon is commonly started by Chrome or a GUI login item, whose PATH
// omits Homebrew and friends. yt-dlp finds ffmpeg through this PATH, so the
// package-manager directories have to survive into the child.
func TestYtdlpEnvAddsBinDirs(t *testing.T) {
	t.Setenv("PATH", "/usr/bin")
	e := &Engine{}
	env := e.ytdlpEnv()

	var path string
	var count int
	for _, kv := range env {
		if strings.HasPrefix(kv, "PATH=") {
			path = strings.TrimPrefix(kv, "PATH=")
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one PATH entry, got %d", count)
	}
	dirs := filepath.SplitList(path)
	if len(dirs) == 0 || dirs[0] != "/usr/bin" {
		t.Errorf("existing PATH must stay first, got %q", path)
	}
	for _, want := range ytdlpBinDirs {
		if !contains(dirs, want) {
			t.Errorf("PATH %q is missing %q", path, want)
		}
	}
}

func TestYtdlpEnvDoesNotDuplicate(t *testing.T) {
	t.Setenv("PATH", strings.Join(ytdlpBinDirs, string(filepath.ListSeparator)))
	e := &Engine{}
	for _, kv := range e.ytdlpEnv() {
		if !strings.HasPrefix(kv, "PATH=") {
			continue
		}
		dirs := filepath.SplitList(strings.TrimPrefix(kv, "PATH="))
		seen := map[string]int{}
		for _, d := range dirs {
			seen[d]++
			if seen[d] > 1 {
				t.Errorf("PATH repeats %q: %v", d, dirs)
			}
		}
	}
}

// A missing yt-dlp must name the right package manager for the platform.
func TestLookupYtdlpErrorMentionsPlatformHint(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	e := &Engine{}
	if _, err := e.lookupTool("yt-dlp"); err != nil {
		if !strings.Contains(err.Error(), ytdlpInstallHint) {
			t.Errorf("error %q does not carry the install hint %q", err, ytdlpInstallHint)
		}
	}
	// If yt-dlp really is installed in one of the fallback dirs the lookup
	// succeeds, which is equally correct — that is the bug this guards.
}

// The fallback is what makes a Homebrew install visible to a daemon started
// with a minimal PATH.
func TestLookupYtdlpFindsBinaryOutsidePath(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "yt-dlp")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	old := ytdlpBinDirs
	ytdlpBinDirs = []string{dir}
	defer func() { ytdlpBinDirs = old }()

	t.Setenv("PATH", t.TempDir()) // empty: forces the fallback
	e := &Engine{}
	got, err := e.lookupTool("yt-dlp")
	if err != nil {
		t.Fatalf("expected the fallback to find %s, got %v", fake, err)
	}
	if got != fake {
		t.Errorf("got %q, want %q", got, fake)
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
