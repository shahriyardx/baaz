package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"

	"baaz"

	"baaz/internal/config"
	"baaz/internal/crx"
	"baaz/internal/daemon"
	"baaz/internal/ipc"
	"baaz/internal/nmhost"
)

// Overridden by -ldflags "-X main.version=..." in release builds.
var version = "dev"

var usage = `baaz — segmented download manager

Usage:
  baaz add URL [--out NAME]     queue a download (starts daemon if needed)
  baaz ls                       list downloads
  baaz pause|resume|cancel ID   control a download
  baaz delete ID                remove a finished download AND its file
  baaz clear                    clear the finished list (files stay)
  baaz on | off                 enable / disable Chrome interception
  baaz config [KEY VALUE]       show or change settings
                                keys: intercept categorize segments
                                      max-active min-size speed-limit dir
  baaz status [--json]          one-shot status (--json = snapshot schema)
  baaz watch                    stream JSON snapshots (for the bar widget)
  baaz daemon                   run the daemon in the foreground
  baaz install-chrome           install the Chrome native-messaging manifest
  baaz fix-chrome               unblock the extension after a manual Remove
` + widgetUsage

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
	case "fix-chrome":
		err = cmdFixChrome()
	case "install-bar", "install-menubar":
		err = cmdInstallWidget()
	case "version", "--version", "-v":
		fmt.Println(version)
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
	limit := "unlimited"
	if s.SpeedLimitKB > 0 {
		limit = fmt.Sprintf("%d KB/s", s.SpeedLimitKB)
	}
	fmt.Printf("intercept   %s\ncategorize  %s\nsegments    %d\nmax-active  %d\nmin-size    %d MB\nspeed-limit %s\ndir         %s\n",
		onOff(s.Intercept), onOff(s.Categorize),
		s.Segments, s.MaxActive, s.MinSizeMB, limit, s.DownloadDir)
	return nil
}

func onOff(v bool) string {
	if v {
		return "on"
	}
	return "off"
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
  "allowed_origins": [%s]
}
`

// The manifest pins a public key, so the extension ID is the same wherever it
// is loaded from — the Web Store, an unpacked folder, or a packed crx.
//
// There are two, because there are two keys. A Store install carries Google's
// key and so gets storeExtID. The crx Linux installs is signed with
// keys/extension-key.pem, which is ours, and keeps selfHostedExtID. The host
// accepts both rather than forcing one install route.
const (
	storeExtID      = "nidklljbjhpljgdeebcpbbnbcijbbcdl"
	selfHostedExtID = "bekhpkepdgjmplfdclkflkkhbpbgeihl"
)

// defaultExtID is what --ext-id defaults to when a single ID is needed.
const defaultExtID = storeExtID

func allowedOrigins(extra string) string {
	ids := []string{storeExtID, selfHostedExtID}
	if extra != "" && extra != storeExtID && extra != selfHostedExtID {
		ids = append(ids, extra)
	}
	quoted := make([]string, 0, len(ids))
	for _, id := range ids {
		quoted = append(quoted, fmt.Sprintf("%q", "chrome-extension://"+id+"/"))
	}
	return strings.Join(quoted, ", ")
}

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
	manifest := fmt.Sprintf(nmManifestTmpl, self, allowedOrigins(*extID))
	for _, dir := range nmHostDirs(home) {
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
	warnTombstones(home)
	return nil
}

// extensionAssets returns the embedded extension tree, its packed crx bytes,
// the extension ID the signing key produces, and the manifest version.
// Each platform installs these differently — see installExtension in
// platform_<goos>.go.
func extensionAssets() (src fs.FS, crxData []byte, id, version string, err error) {
	src, err = fs.Sub(assets.Extension, "extension")
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("embed: %w", err)
	}
	crxData, id, err = crx.Pack(src, assets.ExtensionKey)
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("pack extension: %w", err)
	}
	var m struct {
		Version string `json:"version"`
	}
	raw, _ := fs.ReadFile(src, "manifest.json")
	json.Unmarshal(raw, &m)
	return src, crxData, id, m.Version, nil
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
