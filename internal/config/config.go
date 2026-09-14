package config

import (
	"encoding/json"
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
	Categorize   *bool  `json:"categorize,omitempty"` // pointer: absent = true
	SpeedLimitKB int    `json:"speedLimitKB"`         // total cap in KB/s; 0 = unlimited
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

func (c *Config) CategorizeOn() bool { return c.Categorize == nil || *c.Categorize }

func (c *Config) SetCategorize(v bool) { c.Categorize = &v }

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

// Load reads ~/.config/baaz/config.json, falling back to defaults for any
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
	if c.SpeedLimitKB < 0 {
		c.SpeedLimitKB = 0
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

// Per-platform layout lives in paths_<goos>.go: configDir, DataDir and
// SocketPath differ, everything below is derived and shared.

func JobsDir() string { return filepath.Join(DataDir(), "jobs") }

func LogPath() string { return filepath.Join(DataDir(), "daemon.log") }

func PidPath() string { return filepath.Join(DataDir(), "daemon.pid") }
