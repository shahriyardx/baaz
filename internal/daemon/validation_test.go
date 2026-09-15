package daemon

import (
	"testing"
	"time"

	"baaz/internal/config"
	"baaz/internal/downloader"
	"baaz/internal/ipc"
)

// A mistyped address used to become a queued job that failed a moment later
// with Go's own wording: `unsupported protocol scheme ""`.
func TestCheckURLRejectsWhatCannotBeDownloaded(t *testing.T) {
	bad := map[string]string{
		"not a url":               "http",
		"":                        "no link",
		"   ":                     "no link",
		"ftp://example.com/x.zip": "not supported",
		"file:///etc/passwd":      "not supported",
		"http://":                 "no website",
		"example.com/file.zip":    "http",
	}
	for in, want := range bad {
		got, err := checkURL(in)
		if err == nil {
			t.Errorf("checkURL(%q) = %q, want an error", in, got)
			continue
		}
		if !contains(err.Error(), want) {
			t.Errorf("checkURL(%q) said %q, want something mentioning %q", in, err, want)
		}
	}
}

func TestCheckURLAcceptsRealLinks(t *testing.T) {
	good := []string{
		"https://example.com/file.zip",
		"http://example.com/file.zip",
		"https://example.com:8443/a/b?c=d#e",
		"  https://example.com/file.zip  ", // pasted with whitespace
	}
	for _, in := range good {
		got, err := checkURL(in)
		if err != nil {
			t.Errorf("checkURL(%q) rejected a valid link: %v", in, err)
		}
		if got == "" {
			t.Errorf("checkURL(%q) returned an empty url", in)
		}
	}
}

// No error message may leak Go's internal phrasing to someone who mistyped.
func TestCheckURLErrorsAreReadable(t *testing.T) {
	for _, in := range []string{"not a url", "ftp://x/y", "http://"} {
		_, err := checkURL(in)
		if err == nil {
			t.Fatalf("expected %q to be rejected", in)
		}
		for _, leak := range []string{"unsupported protocol scheme", "parse ", "invalid URI"} {
			if contains(err.Error(), leak) {
				t.Errorf("checkURL(%q) leaked internal wording: %q", in, err)
			}
		}
	}
}

func TestWholeNumberReportsWhatIsWrong(t *testing.T) {
	if _, err := wholeNumber("segments", "999", 1, 32); err == nil {
		t.Error("999 segments should be refused, not silently ignored")
	} else if !contains(err.Error(), "between 1 and 32") {
		t.Errorf("unhelpful message: %v", err)
	}
	if _, err := wholeNumber("segments", "lots", 1, 32); err == nil {
		t.Error("a non-number should be refused")
	}
	if n, err := wholeNumber("segments", " 8 ", 1, 32); err != nil || n != 8 {
		t.Errorf("got %d, %v; want 8 with no error", n, err)
	}
	if n, err := wholeNumber("speed-limit", "0", 0, 1<<20); err != nil || n != 0 {
		t.Errorf("0 must be allowed to lift the cap: %d, %v", n, err)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// The background tool fetch has to reach the UI, or the app looks idle while
// it quietly pulls 80MB.
func TestSnapshotCarriesSetupProgress(t *testing.T) {
	m := &Manager{
		jobs:         map[string]*downloader.Job{},
		cancelIntent: map[string]bool{},
		samples:      map[string][]speedSample{},
		lastSave:     map[string]time.Time{},
		subs:         map[chan *ipc.Snapshot]struct{}{},
		cfg:          &config.Config{},
	}
	if got := m.Snapshot().Setup; got != "" {
		t.Errorf("idle snapshot reported setup %q", got)
	}
	m.setup = "setting up video support — 12.3MB of 35.4MB"
	if got := m.Snapshot().Setup; got != m.setup {
		t.Errorf("Snapshot().Setup = %q, want %q", got, m.setup)
	}
}
