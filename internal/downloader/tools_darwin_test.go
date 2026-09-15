package downloader

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Exercises the real fetch path — direct release download, checksum,
// gunzip, install — against a tiny asset from the same release the ffmpeg
// binary comes from. The binaries themselves are tens of megabytes, which is
// not something to pull on every test run.
func TestFetchToolInstallsFromADirectReleaseURL(t *testing.T) {
	if testing.Short() {
		t.Skip("network test")
	}
	dir := t.TempDir()
	e := &Engine{Client: &http.Client{Timeout: 60 * time.Second}, ToolsDir: dir}
	asset := "darwin-arm64.LICENSE.gz"
	if runtime.GOARCH != "arm64" {
		asset = "darwin-x64.LICENSE.gz"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := e.fetchTool(ctx, fetchSpec{
		url:  releaseURL(ffmpegRepo, asset),
		gz:   true,
		name: "LICENSE",
	}); err != nil {
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
	head := make([]byte, 2)
	f, _ := os.Open(out)
	defer f.Close()
	f.Read(head)
	if head[0] == 0x1f && head[1] == 0x8b {
		t.Error("file is still gzip-compressed")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("leftover files: %v", entries)
	}
}

// The whole point of the rewrite: provisioning must not touch api.github.com,
// whose 60-an-hour-per-IP limit is shared by everyone behind the same
// address and returns 403 once it is gone.
func TestProvisioningNeverCallsTheGitHubAPI(t *testing.T) {
	for _, u := range []string{
		releaseURL(ytdlpRepo, ytdlpAsset),
		releaseURL(ytdlpRepo, ytdlpSums),
		releaseURL(ffmpegRepo, ffmpegAsset()),
	} {
		if strings.Contains(u, "api.github.com") {
			t.Errorf("%s goes through the rate-limited API", u)
		}
		if !strings.HasPrefix(u, "https://github.com/") {
			t.Errorf("%s is not a direct release download", u)
		}
		if !strings.Contains(u, "/releases/latest/download/") {
			t.Errorf("%s is not a latest-release download URL", u)
		}
	}
}

// A refusal has to say what to do about it, not just repeat the status code.
func TestRateLimitedDownloadExplainsItself(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusForbidden)
	}))
	defer srv.Close()

	e := &Engine{Client: srv.Client(), ToolsDir: t.TempDir()}
	_, _, _, err := e.get(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("a 403 must be an error")
	}
	for _, want := range []string{"refused", "brew install"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("message %q does not mention %q", err, want)
		}
	}
}

// A wrong checksum must leave nothing installed.
func TestFetchToolRejectsAMismatchedChecksum(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/SUMS") {
			// A sum for content the asset does not have.
			w.Write([]byte(strings.Repeat("a", 64) + "  tool\n"))
			return
		}
		w.Write([]byte("#!/bin/sh\ntrue\n"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	e := &Engine{Client: srv.Client(), ToolsDir: dir}
	err := e.fetchTool(context.Background(), fetchSpec{
		url:     srv.URL + "/tool",
		sumsURL: srv.URL + "/SUMS",
		sumName: "tool",
		name:    "tool",
	})
	if err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("expected a checksum failure, got %v", err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("a failed fetch left files behind: %v", entries)
	}
}

// A download that arrives intact but cannot run is not installed either — a
// truncated or wrong-architecture binary would otherwise fail much later,
// during someone's download, for no visible reason.
func TestFetchToolRejectsSomethingThatWillNotRun(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("this is not a program"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	e := &Engine{Client: srv.Client(), ToolsDir: dir}
	err := e.fetchTool(context.Background(), fetchSpec{
		url:    srv.URL + "/tool",
		name:   "tool",
		verify: []string{"--version"},
	})
	if err == nil || !strings.Contains(err.Error(), "did not run") {
		t.Fatalf("expected a run check to fail, got %v", err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("left files behind: %v", entries)
	}
}

// And one that does run is installed.
func TestFetchToolInstallsSomethingThatRuns(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("#!/bin/sh\nexit 0\n"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	e := &Engine{Client: srv.Client(), ToolsDir: dir}
	if err := e.fetchTool(context.Background(), fetchSpec{
		url:    srv.URL + "/tool",
		name:   "tool",
		verify: []string{"--version"},
	}); err != nil {
		t.Fatalf("install failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "tool")); err != nil {
		t.Errorf("not installed: %v", err)
	}
}

func TestExpectedSumParsesAChecksumFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(
			"1111111111111111111111111111111111111111111111111111111111111111  other\n" +
				"2222222222222222222222222222222222222222222222222222222222222222  yt-dlp_macos\n"))
	}))
	defer srv.Close()

	e := &Engine{Client: srv.Client()}
	got, err := e.expectedSum(context.Background(), srv.URL, "yt-dlp_macos")
	if err != nil {
		t.Fatal(err)
	}
	if got != strings.Repeat("2", 64) {
		t.Errorf("got %q", got)
	}
	if _, err := e.expectedSum(context.Background(), srv.URL, "absent"); err == nil {
		t.Error("a missing entry must be an error, not an empty sum that skips the check")
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

// The quality menu must not be the thing that downloads 80MB of tools. It
// opens on hover, has nowhere to show progress, and sits inside a timeout —
// so a fresh machine spent up to 45 seconds on "Checking qualities…" in
// silence, then gave up.
func TestQualityLookupDoesNotProvision(t *testing.T) {
	dir := t.TempDir()   // empty: no tools here
	e := NewEngine(1, 1) // and a client that would fail loudly if used
	e.Client = &http.Client{Transport: refusingTransport{t}}
	e.ToolsDir = dir

	// Genuinely hide every copy on this machine, including the package
	// managers' — otherwise the test finds Homebrew's yt-dlp and proves
	// nothing, which is exactly how the silent fetch went unnoticed.
	t.Setenv("PATH", t.TempDir())
	saved := ytdlpBinDirs
	ytdlpBinDirs = nil
	t.Cleanup(func() { ytdlpBinDirs = saved })

	start := time.Now()
	_, err := e.AvailableQualities(context.Background(), "https://youtube.com/watch?v=x")
	if err == nil {
		t.Fatal("expected it to report that the tools are not ready")
	}
	if !errors.Is(err, errToolsNotReady) {
		t.Errorf("got %v, want errToolsNotReady", err)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("took %s — it should answer at once, not go fetching", d)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("it downloaded something: %v", entries)
	}
}

// Any HTTP at all during a quality lookup is the bug this guards against.
type refusingTransport struct{ t *testing.T }

func (r refusingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r.t.Errorf("quality lookup made a network request to %s", req.URL)
	return nil, fmt.Errorf("refused")
}

// The download path is where fetching belongs: it has a row to report on.
// This checks the note is raised before the network is touched, so the row
// says "getting yt-dlp (one time)" rather than sitting blank.
func TestProvisioningAnnouncesItselfBeforeFetching(t *testing.T) {
	saved := ytdlpBinDirs
	ytdlpBinDirs = nil
	t.Cleanup(func() { ytdlpBinDirs = saved })
	t.Setenv("PATH", t.TempDir())

	var notes []string
	e := NewEngine(1, 1)
	e.ToolsDir = t.TempDir()
	// Refuse every request, so the only thing that can reach the caller is
	// the note raised on the way in.
	e.Client = &http.Client{Transport: failingTransport{}}

	err := e.ensureMediaTools(context.Background(), func(s string) { notes = append(notes, s) })
	if err == nil {
		t.Fatal("expected the fetch to fail with no network")
	}
	if len(notes) == 0 {
		t.Fatal("nothing was reported — the download row would sit blank while this happens")
	}
	if !strings.Contains(notes[0], "yt-dlp") || !strings.Contains(notes[0], "one time") {
		t.Errorf("first note was %q, want something naming the tool and saying it is one-off", notes[0])
	}
}

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("no network")
}
