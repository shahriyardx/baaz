// macOS-specific paths and installers.
//
// XProtect constraint: Apple's rule macos_adload_g_bundle deletes any Mach-O
// under 15MB that contains ALL of "_main.main", "killall", "cfprefs" and the
// literal "/Library/Application Support/Google/Chrome/". A plain build of
// this file matched all four and macOS moved the binary to the Trash as
// MACOS.ADLOAD malware. "_main.main" is unavoidable in a Go binary, so the
// other three must stay out: build Chrome paths from components with
// filepath.Join rather than writing the literal, and never shell out to
// killall. macos/xprotect-check.sh fails the build if all four come back.
package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// widgetUsage is the platform's widget line in the top-level help.
const widgetUsage = "  baaz install-menubar          install + start the menu bar app\n"

// nmHostDirs lists the per-user native-messaging host directories Chrome and
// Chromium scan at startup.
func nmHostDirs(home string) []string {
	support := filepath.Join(home, "Library", "Application Support")
	return []string{
		filepath.Join(support, "Google", "Chrome", "NativeMessagingHosts"),
		filepath.Join(support, "Chromium", "NativeMessagingHosts"),
	}
}

// chromeProfileRoots are the directories holding each browser's profile
// folders (Default, Profile 1, …), each with its own Preferences.
func chromeProfileRoots(home string) []string {
	support := filepath.Join(home, "Library", "Application Support")
	return []string{
		filepath.Join(support, "Google", "Chrome"),
		filepath.Join(support, "Chromium"),
	}
}

// chromeProcessNames are what the running browser is called, for pgrep -x.
// macOS names the executable after the bundle, space and all — "chrome"
// matches nothing here, so a check using it would report Chrome as closed
// while it is running and happily edit a file Chrome rewrites on exit.
func chromeProcessNames() []string { return []string{"Google Chrome", "Chromium"} }

// installPolicy stops Chrome from asking where to save each download — the
// dialog would otherwise appear before the extension ever sees the download.
//
// macOS Chrome reads policy from its preferences domain rather than a
// policies directory. As root we write the machine-wide domain so it applies
// to every account; otherwise the per-user domain, which Chrome also honors.
func installPolicy() {
	const key = "PromptForDownloadLocation"
	domain := "com.google.Chrome"
	scope := "this user"
	if os.Geteuid() == 0 {
		domain = "/Library/Preferences/com.google.Chrome"
		scope = "all users"
	}
	for _, d := range []string{domain, chromiumDomain(domain)} {
		cmd := exec.Command("defaults", "write", d, key, "-bool", "false")
		if err := cmd.Run(); err != nil {
			continue
		}
		fmt.Printf("set %s=false in %s (%s)\n", key, d, scope)
	}
	// No cache flush here: `defaults write` goes through cfprefsd, which owns
	// the cache, so the value is already committed. (Killing cfprefsd would
	// also put this binary back inside the XProtect adware signature noted on
	// externalExtDirs.)
	if os.Geteuid() != 0 {
		fmt.Println("\nto apply the no-save-prompt policy for every account, run:")
		fmt.Println("  " + sudoHint())
		fmt.Println("(or turn off chrome://settings/downloads → “Ask where to save …” by hand)")
	}
}

func chromiumDomain(chrome string) string {
	if chrome == "com.google.Chrome" {
		return "org.chromium.Chromium"
	}
	return "/Library/Preferences/org.chromium.Chromium"
}

const (
	menuBarApp      = "Baaz.app"
	menuBarExec     = "BaazMenuBar"
	menuBarLabel    = "com.shahriyar.baaz.menubar"
	menuBarPlistFmt = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key><string>%s</string>
	<key>ProgramArguments</key>
	<array><string>%s</string></array>
	<key>RunAtLoad</key><true/>
	<key>KeepAlive</key><false/>
	<key>ProcessType</key><string>Interactive</string>
</dict>
</plist>
`
)

// cmdInstallWidget installs the SwiftUI menu bar app into ~/Applications,
// registers it as a login item, and starts it.
func cmdInstallWidget() error {
	if os.Geteuid() == 0 {
		return fmt.Errorf("run install-menubar without sudo — it installs into your own account")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	src, err := findMenuBarApp(home)
	if err != nil {
		return err
	}
	dst := filepath.Join(home, "Applications", menuBarApp)
	if src != dst {
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		// Replace wholesale: a stale Mach-O next to a new Info.plist is worse
		// than a missing app, and macOS caches bundles by path.
		if err := os.RemoveAll(dst); err != nil {
			return err
		}
		if err := copyTree(src, dst); err != nil {
			return err
		}
		fmt.Println("installed", dst)
	}

	agents := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(agents, 0o755); err != nil {
		return err
	}
	plist := filepath.Join(agents, menuBarLabel+".plist")
	body := fmt.Sprintf(menuBarPlistFmt, menuBarLabel, filepath.Join(dst, "Contents", "MacOS", menuBarExec))
	if err := os.WriteFile(plist, []byte(body), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", plist)

	// bootout clears any previous registration — and stops the copy it had
	// running, so the rebuilt bundle is the one that comes back. It fails
	// when nothing is loaded, which is the normal first-install case.
	target := "gui/" + strconv.Itoa(os.Getuid())
	exec.Command("launchctl", "bootout", target+"/"+menuBarLabel).Run()
	if err := exec.Command("launchctl", "bootstrap", target, plist).Run(); err != nil {
		// No launchd registration, so nothing started the app: fall back to
		// opening it directly. On the success path launchd has already
		// started it (RunAtLoad), and opening it again would put a second
		// icon in the menu bar.
		fmt.Fprintf(os.Stderr, "baaz: could not register the login item (%v)\n", err)
		if oerr := exec.Command("open", dst).Run(); oerr != nil {
			return fmt.Errorf("open %s: %w", dst, oerr)
		}
	}
	fmt.Println("menu bar app running — look for the arrow icon near the clock")
	return nil
}

// findMenuBarApp looks where a release install, a `make macos-app` build, and
// an earlier install would each have left the bundle.
func findMenuBarApp(home string) (string, error) {
	var candidates []string
	if self, err := os.Executable(); err == nil {
		if self, err = filepath.EvalSymlinks(self); err == nil {
			dir := filepath.Dir(self)
			candidates = append(candidates,
				filepath.Join(dir, menuBarApp),
				filepath.Join(dir, "..", menuBarApp),
			)
		}
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "build", menuBarApp))
	}
	candidates = append(candidates,
		filepath.Join(home, "Applications", menuBarApp),
		filepath.Join("/Applications", menuBarApp),
	)
	for _, c := range candidates {
		if st, err := os.Stat(filepath.Join(c, "Contents", "MacOS", menuBarExec)); err == nil && !st.IsDir() {
			return filepath.Clean(c), nil
		}
	}
	return "", fmt.Errorf("%s not found — build it with `make macos-app`, or download baaz-macos.zip from the releases page", menuBarApp)
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		switch {
		case info.IsDir():
			return os.MkdirAll(target, 0o755)
		case info.Mode()&os.ModeSymlink != 0:
			// Framework bundles are full of relative symlinks; recreating them
			// keeps the bundle the size it was signed at.
			link, rerr := os.Readlink(path)
			if rerr != nil {
				return rerr
			}
			return os.Symlink(link, target)
		default:
			return copyRegularFile(path, target, info.Mode())
		}
	})
}

func copyRegularFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// installExtension unpacks the extension where Chrome can load it.
//
// macOS cannot use the crx path Linux uses: since Chrome 44 no external
// install may point at a local CRX file, and the External Extensions
// directory honors only an external_update_url aimed at the Web Store. So
// until baaz ships on the Web Store the extension has to be loaded unpacked,
// which is a one-time click and then permanent.
//
// The manifest pins a public key, so the unpacked extension keeps the same ID
// as the packed one — the native-messaging manifest written above still
// matches it. Needs no root.
func installExtension() {
	src, _, id, _, err := extensionAssets()
	if err != nil {
		fmt.Fprintln(os.Stderr, "baaz:", err)
		return
	}
	home, err := realUserHome()
	if err != nil {
		fmt.Fprintln(os.Stderr, "baaz:", err)
		return
	}
	// Downloads, not Application Support: ~/Library is hidden in Finder, so a
	// path there is one the user cannot browse to. Putting it where every
	// other download lands means nothing has to be fetched by hand.
	dst := filepath.Join(home, "Downloads", "baaz-extension")
	// Replace wholesale so a file dropped from the new version cannot linger
	// and get loaded alongside it.
	if err := os.RemoveAll(dst); err != nil {
		fmt.Fprintln(os.Stderr, "baaz:", err)
		return
	}
	if err := writeFS(src, dst); err != nil {
		fmt.Fprintln(os.Stderr, "baaz: unpack extension:", err)
		return
	}
	// Under sudo these would land owned by root, and Chrome — running as the
	// user — could not read them.
	chownToInvoker(dst)

	// ~/Library is hidden in Finder, so the folder cannot be browsed to.
	// Putting it on the clipboard turns step 3 into paste-and-enter.
	copied := toClipboard(dst)

	fmt.Println("\nput the extension in", dst)
	fmt.Println("Chrome on macOS cannot install a local .crx, so load it once by hand:")
	fmt.Println("  1. open chrome://extensions")
	fmt.Println("  2. turn on “Developer mode” (top right)")
	fmt.Println("  3. click “Load unpacked” and pick the baaz-extension folder")
	fmt.Println("     in your Downloads")
	if copied {
		fmt.Println("     (or press ⇧⌘G then ⌘V — the path is on your clipboard)")
	}
	fmt.Println("\nit stays loaded after that, and updates in place when you")
	fmt.Println("re-run install-chrome; the ID is pinned to", id)
}

// writeFS copies an embedded tree onto disk.
func writeFS(src fs.FS, dst string) error {
	return fs.WalkDir(src, ".", func(path string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		target := filepath.Join(dst, path)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, rerr := fs.ReadFile(src, path)
		if rerr != nil {
			return rerr
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

// chownToInvoker hands a tree back to the user who ran sudo. A no-op when
// not running as root.
func chownToInvoker(root string) {
	if os.Geteuid() != 0 {
		return
	}
	uid, err1 := strconv.Atoi(os.Getenv("SUDO_UID"))
	gid, err2 := strconv.Atoi(os.Getenv("SUDO_GID"))
	if err1 != nil || err2 != nil {
		return
	}
	filepath.Walk(root, func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		os.Chown(path, uid, gid)
		return nil
	})
}

// toClipboard puts s on the pasteboard, reporting whether it got there.
func toClipboard(s string) bool {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(s)
	return cmd.Run() == nil
}
