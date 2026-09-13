package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// macOS keeps config and state together under Application Support, which is
// where a user (and Time Machine) expects to find them. XDG_* is still
// honored when explicitly set, so a shared dotfile setup keeps working.

const appSupport = "Library/Application Support/baaz"

func configDir() string {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "baaz")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, appSupport)
}

func DataDir() string {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, "baaz")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, appSupport)
}

// SocketPath uses the per-user temp dir ($TMPDIR, /var/folders/…), which is
// private to the user and cleared between boots. It stays well inside the
// 104-byte sun_path limit; $HOME would not for a long user name.
func SocketPath() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("baaz-%d.sock", os.Getuid()))
}
