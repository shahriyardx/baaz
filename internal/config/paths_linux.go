package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Linux follows the XDG base-directory spec.

func configDir() string {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "baaz")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "baaz")
}

func DataDir() string {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, "baaz")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "baaz")
}

func SocketPath() string {
	if d := os.Getenv("XDG_RUNTIME_DIR"); d != "" {
		return filepath.Join(d, "baaz.sock")
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("baaz-%d.sock", os.Getuid()))
}
