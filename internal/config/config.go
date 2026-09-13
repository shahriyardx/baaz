package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Segments     int    `json:"segments"`
	DownloadDir  string `json:"downloadDir"`
	MaxActive    int    `json:"maxActive"`
	MinSplitSize int64  `json:"minSplitSize"`
	Intercept    *bool  `json:"intercept,omitempty"` // pointer: absent = true
	MinSizeMB    int    `json:"minSizeMB"`
}

func Default() *Config {
	t := true
	return &Config{
		Segments:     8,
		DownloadDir:  "~/Downloads",
		MaxActive:    3,
		MinSplitSize: 1 << 20, // 1 MiB
		Intercept:    &t,
		MinSizeMB:    5,
	}
}

func (c *Config) InterceptOn() bool { return c.Intercept == nil || *c.Intercept }

func (c *Config) SetIntercept(v bool) { c.Intercept = &v }

// Save persists the config for the next daemon start; live values are
// applied by the daemon directly.
func (c *Config) Save() error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "config.json"), append(data, '\n'), 0o644)
}

// Load reads ~/.config/dm/config.json, falling back to defaults for any
// missing field or a missing/broken file.
func Load() *Config {
	c := Default()
	path := filepath.Join(configDir(), "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return c
	}
	_ = json.Unmarshal(data, c)
	if c.Segments < 1 {
		c.Segments = 1
	}
	if c.MaxActive < 1 {
		c.MaxActive = 1
	}
	if c.MinSplitSize < 1 {
		c.MinSplitSize = 1 << 20
	}
	if c.DownloadDir == "" {
		c.DownloadDir = "~/Downloads"
	}
	return c
}

func (c *Config) ResolvedDownloadDir() string {
	return ExpandHome(c.DownloadDir)
}

func ExpandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}

func configDir() string {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "dm")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "dm")
}

func DataDir() string {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, "dm")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "dm")
}

func JobsDir() string { return filepath.Join(DataDir(), "jobs") }

func LogPath() string { return filepath.Join(DataDir(), "daemon.log") }

func PidPath() string { return filepath.Join(DataDir(), "daemon.pid") }

func SocketPath() string {
	if d := os.Getenv("XDG_RUNTIME_DIR"); d != "" {
		return filepath.Join(d, "dm.sock")
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("dm-%d.sock", os.Getuid()))
}
