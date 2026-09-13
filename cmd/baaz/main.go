package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"baaz"

	"baaz/internal/config"
	"baaz/internal/crx"
	"baaz/internal/daemon"
	"baaz/internal/ipc"
	"baaz/internal/nmhost"
)

const usage = `baaz — segmented download manager

Usage:
  baaz add URL [--out NAME]     queue a download (starts daemon if needed)
  baaz ls                       list downloads
  baaz pause|resume|cancel ID   control a download
  baaz delete ID                remove a finished download AND its file
  baaz clear                    clear the finished list (files stay)
  baaz on | off                 enable / disable Chrome interception
  baaz config [KEY VALUE]       show or change settings
                              keys: intercept segments max-active min-size dir
  baaz status [--json]          one-shot status (--json = snapshot schema)
  baaz watch                    stream JSON snapshots (for the bar widget)
  baaz daemon                   run the daemon in the foreground
  baaz install-chrome           install the Chrome native-messaging manifest
  baaz install-bar              install + enable the Omarchy bar widget
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(2)
	}
	// Chrome launches the NM host as `<path> chrome-extension://<id>/ ...`;
	// the host manifest cannot pass arguments, so detect the origin argv.
	if strings.HasPrefix(os.Args[1], "chrome-extension://") {
		if err := nmhost.Run(); err != nil {
			log.Fatal(err)
		}
		return
	}

	var err error
	switch os.Args[1] {
	case "daemon":
		err = runDaemon()
	case "nm-host":
		err = nmhost.Run()
	case "add":
		err = cmdAdd(os.Args[2:])
	case "ls":
		err = cmdStatus(false)
	case "status":
		err = cmdStatus(len(os.Args) > 2 && os.Args[2] == "--json")
	case "watch":
		err = cmdWatch()
	case "pause", "resume", "cancel", "delete":
		err = cmdControl(os.Args[1], os.Args[2:])
	case "clear":
		err = cmdClear()
	case "on":
		err = cmdConfig([]string{"intercept", "true"})
	case "off":
		err = cmdConfig([]string{"intercept", "false"})
	case "config":
		err = cmdConfig(os.Args[2:])
	case "install-chrome":
		err = cmdInstallChrome(os.Args[2:])
	case "install-bar":
		err = cmdInstallBar()
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Print(usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "baaz:", err)
		os.Exit(1)
	}
}

func runDaemon() error {
	if err := os.MkdirAll(config.DataDir(), 0o755); err != nil {
		return err
	}
	// Single-instance lock. Losing the race to a live daemon is success.
	pidf, err := os.OpenFile(config.PidPath(), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	if err := syscall.Flock(int(pidf.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if c, derr := ipc.Dial(config.SocketPath(), config.LogPath(), false); derr == nil {
			c.Close()
			log.Println("daemon already running; exiting")
			return nil
		}
		return fmt.Errorf("another daemon holds the lock but its socket is dead")
	}
	pidf.Truncate(0)
	fmt.Fprintf(pidf, "%d\n", os.Getpid())

	cfg := config.Load()
	mgr := daemon.NewManager(cfg)
	srv, err := ipc.NewServer(config.SocketPath(), mgr)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		log.Println("shutting down")
		mgr.Shutdown()
		cancel()
	}()

	mgr.Start(ctx)
	log.Println("baaz daemon listening on", config.SocketPath())
	defer os.Remove(config.SocketPath())
	return srv.Serve(ctx)
}

func dial() (*ipc.Client, error) {
	return ipc.Dial(config.SocketPath(), config.LogPath(), true)
}

func cmdAdd(args []string) error {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	out := fs.String("out", "", "output filename")
	format := fs.String("format", "", "media quality: best|1080|720|480|audio")
	fs.Parse(args)
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: baaz add URL [--out NAME] [--format QUALITY]")
	}
	c, err := dial()
	if err != nil {
		return err
	}
	defer c.Close()
	resp, err := c.Do(&ipc.Request{Cmd: "add", URL: fs.Arg(0), Filename: *out, Format: *format})
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}
	fmt.Println(resp.ID)
	return nil
}

func cmdControl(cmd string, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: baaz %s ID", cmd)
	}
	c, err := dial()
	if err != nil {
		return err
	}
	defer c.Close()
	resp, err := c.Do(&ipc.Request{Cmd: cmd, ID: args[0]})
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

func cmdClear() error {
	c, err := dial()
	if err != nil {
		return err
	}
	defer c.Close()
	resp, err := c.Do(&ipc.Request{Cmd: "clear"})
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

func cmdConfig(args []string) error {
	c, err := dial()
	if err != nil {
		return err
	}
	defer c.Close()
	req := &ipc.Request{Cmd: "getconfig"}
	if len(args) == 2 {
		req = &ipc.Request{Cmd: "setconfig", Settings: map[string]string{args[0]: args[1]}}
	} else if len(args) != 0 {
		return fmt.Errorf("usage: baaz config [KEY VALUE]")
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}
	s := resp.Snapshot.Settings
	state := "off"
	if s.Intercept {
		state = "on"
	}
	fmt.Printf("intercept   %s\nsegments    %d\nmax-active  %d\nmin-size    %d MB\ndir         %s\n",
		state, s.Segments, s.MaxActive, s.MinSizeMB, s.DownloadDir)
	return nil
}

func cmdStatus(asJSON bool) error {
	c, err := dial()
	if err != nil {
		return err
	}
	defer c.Close()
	resp, err := c.Do(&ipc.Request{Cmd: "status"})
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}
	if asJSON {
		return json.NewEncoder(os.Stdout).Encode(resp.Snapshot)
	}
	snap := resp.Snapshot
	if len(snap.Jobs) == 0 && len(snap.Recent) == 0 {
		fmt.Println("no downloads")
		return nil
	}
	for _, j := range snap.Jobs {
		fmt.Printf("%-8s  %-7s  %6s  %9s/s  %s\n",
			j.ID, j.State, percent(j.Done, j.Total), human(j.Speed), j.Name)
	}
	for _, j := range snap.Recent {
		fmt.Printf("%-8s  %-7s  %6s  %11s  %s\n",
			j.ID, j.State, "100%", human(j.Total), j.Name)
	}
	return nil
}

func cmdWatch() error {
	c, err := dial()
	if err != nil {
		return err
	}
	defer c.Close()
	out := os.Stdout
	return c.Watch(func(line []byte) bool {
		out.Write(append(line, '\n'))
		return true
	})
}

func percent(done, total int64) string {
	if total <= 0 {
		return "?"
	}
	return fmt.Sprintf("%d%%", done*100/total)
}

func human(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1fGB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1fMB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1fKB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%dB", n)
	}
}

const nmManifestTmpl = `{
  "name": "com.shahriyar.baaz",
  "description": "baaz download manager",
  "path": "%s",
  "type": "stdio",
  "allowed_origins": ["chrome-extension://%s/"]
}
`

// The extension manifest pins a public key, which makes the extension ID
// deterministic no matter where or how it is loaded.
const defaultExtID = "bekhpkepdgjmplfdclkflkkhbpbgeihl"

func cmdInstallChrome(args []string) error {
	fs := flag.NewFlagSet("install-chrome", flag.ExitOnError)
	extID := fs.String("ext-id", defaultExtID, "Chrome extension ID override")
	fs.Parse(args)
	self, err := os.Executable()
	if err != nil {
		return err
	}
	self, err = filepath.EvalSymlinks(self)
	if err != nil {
		return err
	}
	home, err := realUserHome()
	if err != nil {
		return err
	}
	manifest := fmt.Sprintf(nmManifestTmpl, self, *extID)
	for _, dir := range []string{
		filepath.Join(home, ".config", "google-chrome", "NativeMessagingHosts"),
		filepath.Join(home, ".config", "chromium", "NativeMessagingHosts"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		path := filepath.Join(dir, "com.shahriyar.baaz.json")
		if err := os.WriteFile(path, []byte(manifest), 0o644); err != nil {
			return err
		}
		fmt.Println("wrote", path)
	}
	fmt.Println("restart Chrome to pick up the native messaging host")
	installPolicy()
	installExtension()
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
	src, err := fs.Sub(assets.Extension, "extension")
	if err != nil {
		fmt.Fprintln(os.Stderr, "dm: embed:", err)
		return
	}
	data, id, err := crx.Pack(src, assets.ExtensionKey)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dm: pack extension:", err)
		return
	}
	var m struct {
		Version string `json:"version"`
	}
	raw, _ := fs.ReadFile(src, "manifest.json")
	json.Unmarshal(raw, &m)

	crxPath := "/usr/share/baaz/baaz.crx"
	if err := os.MkdirAll(filepath.Dir(crxPath), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "baaz:", err)
		return
	}
	if err := os.WriteFile(crxPath, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "baaz:", err)
		return
	}
	fmt.Println("wrote", crxPath)

	entry := fmt.Sprintf("{ \"external_crx\": %q, \"external_version\": %q }\n", crxPath, m.Version)
	for _, dir := range []string{
		"/usr/share/google-chrome/extensions",
		"/usr/share/chromium/extensions",
	} {
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

// sudoHint prints the sudo re-run command with an absolute path, because
// sudo's secure_path does not include ~/.local/bin.
func sudoHint() string {
	self, err := os.Executable()
	if err != nil {
		return "sudo baaz install-chrome"
	}
	return "sudo " + self + " install-chrome"
}

const barPluginID = "shahriyardx.baaz"

// cmdInstallBar copies the embedded Omarchy bar widget into the user's
// plugin directory and enables it — the shell only discovers plugins there.
func cmdInstallBar() error {
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

// realUserHome resolves the invoking user's home even under sudo, so
// `sudo baaz install-chrome` still writes Chrome files into the right place.
func realUserHome() (string, error) {
	if su := os.Getenv("SUDO_USER"); su != "" && os.Geteuid() == 0 {
		if u, err := user.Lookup(su); err == nil {
			return u.HomeDir, nil
		}
	}
	return os.UserHomeDir()
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
