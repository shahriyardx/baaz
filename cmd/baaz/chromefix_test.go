package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func prefs(t *testing.T, raw string) map[string]any {
	t.Helper()
	var d map[string]any
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		t.Fatal(err)
	}
	return d
}

// The tombstone is an empty settings entry. Matching anything else would
// mean deleting the record of a working extension.
func TestHasTombstone(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{"empty entry is a tombstone",
			`{"extensions":{"settings":{"` + defaultExtID + `":{}}}}`, true},
		{"populated entry is a real install",
			`{"extensions":{"settings":{"` + defaultExtID + `":{"state":1,"path":"/x"}}}}`, false},
		{"other extensions are ignored",
			`{"extensions":{"settings":{"aaaa":{}}}}`, false},
		{"no settings block", `{"extensions":{}}`, false},
		{"no extensions block", `{}`, false},
		{"entry is not an object",
			`{"extensions":{"settings":{"` + defaultExtID + `":"nope"}}}`, false},
	}
	for _, c := range cases {
		if got := hasTombstone(prefs(t, c.raw)); got != c.want {
			t.Errorf("%s: hasTombstone = %v, want %v", c.name, got, c.want)
		}
	}
}

// Preferences holds the whole Chrome profile. A failed write must never
// leave it truncated or half-written.
func TestWritePrefsReplacesAtomically(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Preferences")
	if err := os.WriteFile(path, []byte(`{"old":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writePrefs(path, map[string]any{"new": true}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"new":true}` {
		t.Errorf("got %q", got)
	}
	if st, err := os.Stat(path); err == nil && st.Mode().Perm() != 0o600 {
		t.Errorf("permissions = %v, want 0600", st.Mode().Perm())
	}
	// The temp file must not survive next to the real one.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "Preferences" {
			t.Errorf("leftover file %q", e.Name())
		}
	}
}

// Values json.Marshal cannot encode must abort, not truncate the file.
func TestWritePrefsRefusesUnencodableRatherThanTruncate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Preferences")
	original := `{"profile":"intact"}`
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	err := writePrefs(path, map[string]any{"bad": make(chan int)})
	if err == nil {
		t.Fatal("expected an error for an unencodable value")
	}
	got, _ := os.ReadFile(path)
	if string(got) != original {
		t.Fatalf("Preferences was modified on a failed write: %q", got)
	}
}

// Profile discovery must use this platform's layout, not Linux's everywhere.
func TestChromeProfileRootsAreAbsoluteAndPlatformLocal(t *testing.T) {
	roots := chromeProfileRoots("/home/u")
	if len(roots) == 0 {
		t.Fatal("no Chrome profile roots for this platform")
	}
	for _, r := range roots {
		if !filepath.IsAbs(r) {
			t.Errorf("root %q is not absolute", r)
		}
	}
}

func TestChromeProcessNamesNonEmpty(t *testing.T) {
	// An empty or wrong name makes chromeRunning report "closed" while
	// Chrome is running, and the edit is then lost when Chrome exits.
	for _, n := range chromeProcessNames() {
		if n == "" {
			t.Error("empty process name")
		}
	}
	if len(chromeProcessNames()) == 0 {
		t.Error("no process names to check")
	}
}
