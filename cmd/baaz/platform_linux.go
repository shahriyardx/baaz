package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"baaz"
)

// widgetUsage is the platform's widget line in the top-level help.
const widgetUsage = "  baaz install-bar              install + enable the Omarchy bar widget\n"

// nmHostDirs lists the per-user native-messaging host directories Chrome and
// Chromium scan at startup.
func nmHostDirs(home string) []string {
	return []string{
		filepath.Join(home, ".config", "google-chrome", "NativeMessagingHosts"),
		filepath.Join(home, ".config", "chromium", "NativeMessagingHosts"),
	}
}

// chromeProfileRoots are the directories holding each browser's profile
// folders (Default, Profile 1, …), each with its own Preferences.
func chromeProfileRoots(home string) []string {
	return []string{
		filepath.Join(home, ".config", "google-chrome"),
		filepath.Join(home, ".config", "chromium"),
	}
}

// chromeProcessNames are what the running browser is called, for pgrep -x.
func chromeProcessNames() []string { return []string{"chrome", "chromium"} }

// crxInstallPath is where the packed extension lands for Chrome to read.
func crxInstallPath() string { return "/usr/share/baaz/baaz.crx" }

// externalExtDirs are the system-wide "external extension" registries; a
// json file dropped here installs the crx on the next browser start.
func externalExtDirs() []string {
	return []string{
		"/usr/share/google-chrome/extensions",
		"/usr/share/chromium/extensions",
	}
}

// installPolicy writes a managed policy that stops Chrome from asking where
// to save each download — the dialog would otherwise appear before the
// extension ever sees the download. Needs root; prints the command when run
// without it.
func installPolicy() {
	const policy = `{ "PromptForDownloadLocation": false }` + "\n"
	dirs := []string{
		"/etc/opt/chrome/policies/managed",
		"/etc/chromium/policies/managed",
	}
	var failed []string
	for _, dir := range dirs {
		path := filepath.Join(dir, "baaz-no-save-prompt.json")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			failed = append(failed, dir)
			continue
		}
		if err := os.WriteFile(path, []byte(policy), 0o644); err != nil {
			failed = append(failed, dir)
			continue
		}
		fmt.Println("wrote", path)
	}
	if len(failed) > 0 {
		fmt.Println("\nto also disable Chrome's save-location dialog system-wide, run:")
		fmt.Println("  " + sudoHint())
		fmt.Println("(or turn off chrome://settings/downloads → “Ask where to save …” by hand)")
	}
}

const barPluginID = "shahriyardx.baaz"

// cmdInstallWidget copies the embedded Omarchy bar widget into the user's
// plugin directory and enables it — the shell only discovers plugins there.
func cmdInstallWidget() error {
	home, err := realUserHome()
	if err != nil {
		return err
	}
	dst := filepath.Join(home, ".config", "omarchy", "plugins", barPluginID)
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	src, err := fs.Sub(assets.BarPlugin, "bar-plugin")
	if err != nil {
		return err
	}
	err = fs.WalkDir(src, ".", func(path string, d fs.DirEntry, werr error) error {
		if werr != nil || d.IsDir() {
			return werr
		}
		data, rerr := fs.ReadFile(src, path)
		if rerr != nil {
			return rerr
		}
		return os.WriteFile(filepath.Join(dst, path), data, 0o644)
	})
	if err != nil {
		return err
	}
	fmt.Println("installed", dst)
	for _, cmdline := range [][]string{
		{"omarchy-shell", "shell", "rescanPlugins"},
		{"omarchy", "plugin", "enable", barPluginID},
		{"omarchy", "bar", "move", barPluginID, "--section", "right"},
	} {
		c := exec.Command(cmdline[0], cmdline[1:]...)
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		if err := c.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "baaz: %s failed (%v) — run it by hand\n", strings.Join(cmdline, " "), err)
		}
		time.Sleep(time.Second) // rescan is async; give the shell a beat
	}
	return nil
}

// installExtension packs the embedded extension into a crx and registers it
// as a Chrome "external extension", so a restart installs it — no unpacked
// loading, no developer mode. Root only (the registry dirs live in /usr).
func installExtension() {
	if os.Geteuid() != 0 {
		fmt.Println("\nrun with sudo to also auto-install the extension into Chrome:")
		fmt.Println("  " + sudoHint())
		return
	}
	_, data, id, version, err := extensionAssets()
	if err != nil {
		fmt.Fprintln(os.Stderr, "baaz:", err)
		return
	}

	crxPath := crxInstallPath()
	if err := os.MkdirAll(filepath.Dir(crxPath), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "baaz:", err)
		return
	}
	if err := os.WriteFile(crxPath, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "baaz:", err)
		return
	}
	fmt.Println("wrote", crxPath)

	entry := fmt.Sprintf("{ \"external_crx\": %q, \"external_version\": %q }\n", crxPath, version)
	for _, dir := range externalExtDirs() {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			continue
		}
		path := filepath.Join(dir, id+".json")
		if err := os.WriteFile(path, []byte(entry), 0o644); err != nil {
			continue
		}
		fmt.Println("wrote", path)
	}
	fmt.Println("extension", id, "installs on next Chrome start (confirm the one-time “Enable” prompt)")
}
