package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// When a user clicks Remove on an externally-installed extension, Chrome
// plants an empty record ("<id>": {}) in that profile's Preferences and
// silently refuses to ever auto-install the id again — no UI, no error.
// These helpers detect that tombstone and remove it on request.

func chromePrefFiles(home string) []string {
	var out []string
	for _, base := range []string{"google-chrome", "chromium"} {
		root := filepath.Join(home, ".config", base)
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			if name != "Default" && !strings.HasPrefix(name, "Profile") {
				continue
			}
			p := filepath.Join(root, name, "Preferences")
			if _, err := os.Stat(p); err == nil {
				out = append(out, p)
			}
		}
	}
	return out
}

func readPrefs(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var d map[string]any
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}
	return d, nil
}

// hasTombstone reports whether the prefs block the extension: the settings
// entry exists but is an empty object.
func hasTombstone(d map[string]any) bool {
	ext, _ := d["extensions"].(map[string]any)
	settings, _ := ext["settings"].(map[string]any)
	entry, ok := settings[defaultExtID].(map[string]any)
	return ok && len(entry) == 0
}

// warnTombstones is called after install-chrome so the block is visible
// instead of failing silently on the next Chrome start.
func warnTombstones(home string) {
	var blocked []string
	for _, p := range chromePrefFiles(home) {
		if d, err := readPrefs(p); err == nil && hasTombstone(d) {
			blocked = append(blocked, filepath.Base(filepath.Dir(p)))
		}
	}
	if len(blocked) == 0 {
		return
	}
	fmt.Println()
	fmt.Println("WARNING: Chrome is blocking the baaz extension in profile(s):", strings.Join(blocked, ", "))
	fmt.Println("(this happens after clicking \"Remove\" on it — Chrome refuses to reinstall it silently)")
	fmt.Println("To fix: close Chrome completely (chrome://quit), then run:")
	fmt.Println("  baaz fix-chrome")
}

func chromeRunning() bool {
	for _, name := range []string{"chrome", "chromium"} {
		if exec.Command("pgrep", "-x", name).Run() == nil {
			return true
		}
	}
	return false
}

// cmdFixChrome removes the tombstone from every profile, so the next Chrome
// start installs the extension fresh (one "Enable" confirmation).
func cmdFixChrome() error {
	if chromeRunning() {
		return fmt.Errorf("close Chrome first (type chrome://quit in the address bar), then run this again")
	}
	home, err := realUserHome()
	if err != nil {
		return err
	}
	fixed := 0
	for _, p := range chromePrefFiles(home) {
		d, err := readPrefs(p)
		if err != nil || !hasTombstone(d) {
			continue
		}
		if err := os.WriteFile(p+".baaz-backup", mustJSON(d), 0o600); err != nil {
			return err
		}
		ext := d["extensions"].(map[string]any)
		delete(ext["settings"].(map[string]any), defaultExtID)
		if prot, ok := d["protection"].(map[string]any); ok {
			if macs, ok := prot["macs"].(map[string]any); ok {
				if e, ok := macs["extensions"].(map[string]any); ok {
					if s, ok := e["settings"].(map[string]any); ok {
						delete(s, defaultExtID)
					}
					if s, ok := e["settings_encrypted_hash"].(map[string]any); ok {
						delete(s, defaultExtID)
					}
				}
			}
		}
		if ucd, ok := d["updateclientdata"].(map[string]any); ok {
			if apps, ok := ucd["apps"].(map[string]any); ok {
				delete(apps, defaultExtID)
			}
		}
		if err := os.WriteFile(p, mustJSON(d), 0o600); err != nil {
			return err
		}
		fmt.Println("unblocked", filepath.Base(filepath.Dir(p)), "(backup: Preferences.baaz-backup)")
		fixed++
	}
	if fixed == 0 {
		fmt.Println("no blocks found — nothing to fix")
		return nil
	}
	fmt.Println("start Chrome and confirm the one-time “Enable” prompt")
	return nil
}

func mustJSON(d map[string]any) []byte {
	data, _ := json.Marshal(d)
	return data
}
