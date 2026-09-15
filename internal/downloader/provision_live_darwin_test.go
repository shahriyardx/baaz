package downloader

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The check that would have caught the 403: provision both real tools into
// an empty directory, the way a Mac with neither installed does on its first
// video download, then run what arrives.
//
// Skipped by default — it pulls tens of megabytes. Run it deliberately:
//
//	go test ./internal/downloader -run TestFreshMachineGetsBothTools -tags= -v -provision
func TestFreshMachineGetsBothTools(t *testing.T) {
	if os.Getenv("BAAZ_PROVISION_TEST") == "" {
		t.Skip("set BAAZ_PROVISION_TEST=1 to download both tools for real")
	}
	dir := t.TempDir()
	e := NewEngine(8, 1<<20)
	e.Client = &http.Client{Timeout: 5 * time.Minute}
	e.ToolsDir = dir

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Fetch both directly rather than through ensureMediaTools: that checks
	// PATH first and would find this machine's own Homebrew copies, which is
	// exactly how the broken download path went unnoticed.
	start := time.Now()
	if err := e.fetchTool(ctx, fetchSpec{
		url:     releaseURL(ytdlpRepo, ytdlpAsset),
		sumsURL: releaseURL(ytdlpRepo, ytdlpSums),
		sumName: ytdlpAsset,
		name:    "yt-dlp",
		verify:  []string{"--version"},
	}); err != nil {
		t.Fatalf("yt-dlp could not be fetched by a machine without it: %v", err)
	}
	if err := e.fetchTool(ctx, fetchSpec{
		url:    releaseURL(ffmpegRepo, ffmpegAsset()),
		gz:     true,
		name:   "ffmpeg",
		verify: []string{"-version"},
	}); err != nil {
		t.Fatalf("ffmpeg could not be fetched by a machine without it: %v", err)
	}
	t.Logf("both fetched in %s", time.Since(start).Round(time.Second))

	for tool, arg := range map[string]string{"yt-dlp": "--version", "ffmpeg": "-version"} {
		p := filepath.Join(dir, tool)
		st, err := os.Stat(p)
		if err != nil {
			t.Errorf("%s was not installed: %v", tool, err)
			continue
		}
		if st.Mode()&0o111 == 0 {
			t.Errorf("%s is not executable", tool)
		}
		out, err := exec.CommandContext(ctx, p, arg).CombinedOutput()
		if err != nil {
			t.Errorf("%s did not run: %v", tool, err)
			continue
		}
		t.Logf("%-7s %.1f MB  %s", tool, float64(st.Size())/1048576,
			strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0])
	}
}

// The note has to move. It used to be one static string for the whole
// download, which on a slow line is indistinguishable from being stuck.
func TestSetupProgressActuallyMoves(t *testing.T) {
	if os.Getenv("BAAZ_PROVISION_TEST") == "" {
		t.Skip("set BAAZ_PROVISION_TEST=1 to download a real tool")
	}
	e := NewEngine(1, 1)
	e.Client = &http.Client{Timeout: 5 * time.Minute}
	e.ToolsDir = t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var notes []string
	if err := e.fetchTool(ctx, fetchSpec{
		url:     releaseURL(ytdlpRepo, ytdlpAsset),
		sumsURL: releaseURL(ytdlpRepo, ytdlpSums),
		sumName: ytdlpAsset,
		name:    "yt-dlp",
		verify:  []string{"--version"},
		note:    func(s string) { notes = append(notes, s) },
	}); err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(notes) < 2 {
		t.Fatalf("only %d updates for a 35MB download — the row would look frozen: %v",
			len(notes), notes)
	}
	seen := map[string]bool{}
	for _, n := range notes {
		seen[n] = true
	}
	if len(seen) < 2 {
		t.Errorf("every update said the same thing: %q", notes[0])
	}
	if !strings.Contains(notes[0], "of ") {
		t.Errorf("no total shown, so there is no sense of how long: %q", notes[0])
	}
	t.Logf("%d updates, first %q, last %q", len(notes), notes[0], notes[len(notes)-1])
}
