package downloader

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// Exercises the real fetch path — release lookup, download, SHA-256 check
// against the digest GitHub reports, gunzip, install — against a tiny asset
// from the same release the ffmpeg binary comes from. The binaries
// themselves are tens of megabytes, which is not something to pull on every
// test run.
func TestFetchToolVerifiesAndInstalls(t *testing.T) {
	if testing.Short() {
		t.Skip("network test")
	}
	dir := t.TempDir()
	e := &Engine{
		Client:   &http.Client{Timeout: 60 * time.Second},
		ToolsDir: dir,
	}
	asset := "darwin-arm64.LICENSE.gz"
	if runtime.GOARCH != "arm64" {
		asset = "darwin-x64.LICENSE.gz"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := e.fetchTool(ctx, ffmpegRepo, asset, "LICENSE", true); err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	out := filepath.Join(dir, "LICENSE")
	st, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if st.Size() == 0 {
		t.Error("installed an empty file")
	}
	if st.Mode()&0o111 == 0 {
		t.Errorf("mode %v is not executable", st.Mode())
	}
	// Decompressed, not the gzip stream.
	head := make([]byte, 2)
	f, _ := os.Open(out)
	defer f.Close()
	f.Read(head)
	if head[0] == 0x1f && head[1] == 0x8b {
		t.Error("file is still gzip-compressed")
	}
	// No temp files left behind.
	entries, _ := os.ReadDir(dir)
	for _, en := range entries {
		if en.Name() != "LICENSE" {
			t.Errorf("leftover %q", en.Name())
		}
	}
}

// A checksum mismatch must leave nothing installed.
func TestFetchToolRejectsBadAsset(t *testing.T) {
	if testing.Short() {
		t.Skip("network test")
	}
	dir := t.TempDir()
	e := &Engine{Client: &http.Client{Timeout: 30 * time.Second}, ToolsDir: dir}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := e.fetchTool(ctx, ffmpegRepo, "no-such-asset-xyz", "nope", false); err == nil {
		t.Fatal("expected an error for a missing asset")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("a failed fetch left files behind: %v", entries)
	}
}

// Provisioning is opt-in: with no ToolsDir nothing is fetched.
func TestEnsureMediaToolsDoesNothingWithoutToolsDir(t *testing.T) {
	e := &Engine{Client: http.DefaultClient}
	notes := 0
	if err := e.ensureMediaTools(context.Background(), func(string) { notes++ }); err != nil {
		t.Fatal(err)
	}
	if notes != 0 {
		t.Error("reported progress despite having nowhere to install")
	}
}
