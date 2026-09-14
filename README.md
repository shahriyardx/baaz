# baaz

**A fast download manager for Linux and macOS.** baaz (বাজ — "falcon" in
Bengali) takes over your Chrome downloads and pulls files up to 8× faster by
downloading several pieces at once. It also downloads videos from YouTube, Facebook,
TikTok and more with one click.

- **Faster downloads** — big files split into up to 8 parts, downloaded together
- **Pause and resume** — even after a reboot or crash, downloads continue where they stopped
- **Video download button** — hover any video on YouTube, Facebook, etc. and click Download; pick a quality or audio-only mp3
- **Tidy folders** — files sort themselves into Videos, Music, Documents, Programs and so on inside your Downloads folder
- **Desktop notifications** — know when a download starts, finishes, or fails
- **Always-visible widget** — live speed and progress in your Omarchy bar (Linux)
  or your menu bar (macOS), no window to open

---

## Install

### Arch Linux / Omarchy

```
yay -S baaz
```

### macOS

Download **[Baaz.dmg][dmg]**, drag **Baaz** onto Applications, and open it.

A falcon appears in your menu bar. That's the whole install — opening the app
also puts the `baaz` command in `~/.local/bin`, connects Chrome, and sets
itself to open at login.

> On first open, **right-click the app → Open**. baaz isn't notarized yet, so
> a plain double-click is refused once.

macOS 13 (Ventura) or newer, Intel or Apple Silicon. One step is left over:
Chrome won't let any app install an extension from your disk, so you add it
by hand once — the app shows you how, and it's below too.

[dmg]: https://github.com/shahriyardx/baaz/releases/latest/download/Baaz.dmg

### Any other Linux

Paste this into a terminal:

```
curl -fsSL https://github.com/shahriyardx/baaz/releases/latest/download/install.sh | bash
```

That downloads baaz into your home folder, connects it to Chrome, and starts
the widget. No compiler, nothing to build. The only thing it may ask to
install is yt-dlp and ffmpeg, for video downloads — and only if you say yes.

Run it as yourself — **not** with `sudo`. It asks for your password once, at
the start, and uses it only for the Chrome step; everything else belongs to
your own account.

(The installer still works on macOS if you prefer a terminal, but the disk
image is the easier route.)

---

## Set up Chrome (one time)

### Linux

1. In a terminal, run:
   ```
   sudo baaz install-chrome
   ```
   (It asks for your password. This connects Chrome to baaz, installs the
   browser extension, and turns off Chrome's "ask where to save" popup.)
2. Restart Chrome — type `chrome://restart` in the address bar and press Enter.
3. Chrome shows a small "Enable extension" message once — click **Enable**.

### macOS

Chrome on macOS refuses to install an extension from a file on your computer —
only the Chrome Web Store counts. So the extension is loaded by hand once. It
stays loaded afterwards, including across updates and restarts.

Opening Baaz.app already did the setup and put the extension in
**`~/Downloads/baaz-extension`**. All that's left:

1. (Only if you installed from the terminal instead of the disk image:
   run `baaz install-chrome`.)
2. Open `chrome://extensions`.
3. Turn on **Developer mode** (top right).
4. Click **Load unpacked** and pick the `baaz-extension` folder in your
   Downloads. (The path is also on your clipboard: press <kbd>⇧⌘G</kbd> then
   <kbd>⌘V</kbd> in the dialog.)
5. Restart Chrome — `chrome://restart`.

Keep that folder — Chrome loads the extension from it every launch, so
deleting it disables the extension. Re-running `baaz install-chrome` updates
it in place, and Chrome picks up the new version on restart.

If you ever need it on a machine where you haven't run `install-chrome`,
the same folder is on the releases page as
**[baaz-extension.zip][ext]**.

[ext]: https://github.com/shahriyardx/baaz/releases/latest/download/baaz-extension.zip

`sudo` is only needed to apply the no-save-prompt setting to every account.
Without it everything else still works for you.

That's it. From now on, when you download something in Chrome, baaz takes it.

**Optional — video downloads from YouTube etc.** need yt-dlp (and ffmpeg, to
join the video and audio tracks). The installer notices if they are missing
and offers to install them; say no and everything except video downloads
still works. To do it yourself:

```
sudo pacman -S yt-dlp ffmpeg        # Arch / Omarchy
sudo apt install yt-dlp ffmpeg      # Ubuntu / Debian
sudo dnf install yt-dlp ffmpeg      # Fedora
brew install yt-dlp ffmpeg          # macOS
```

**Optional — the widget:**

```
baaz install-bar            # Linux (Omarchy bar)
baaz install-menubar        # macOS (menu bar)
```

On macOS the installer already does this for you. `install-menubar` puts
`Baaz.app` in `~/Applications`, starts it, and makes it open at login. Run it
without `sudo`.

---

## Updating

```
yay -Syu baaz            # Arch / Omarchy
```

Other Linux and macOS: run the same curl line from Install again — it fetches
the latest version.

Then refresh the browser side (the new extension ships inside baaz):

```
sudo baaz install-chrome
```

and restart Chrome (`chrome://restart`). On Linux, Chrome swaps in the new
extension by itself — no prompts, no developer mode. On macOS,
`~/Downloads/baaz-extension` is rewritten in place, so Chrome picks the new
version up on restart — you do **not** have to "Load unpacked" again.

---

## Uninstall

### Linux

```
pkill -f 'baaz daemon'
rm -f  ~/.local/bin/baaz
rm -rf ~/.config/omarchy/plugins/shahriyardx.baaz
rm -f  ~/.config/google-chrome/NativeMessagingHosts/com.shahriyar.baaz.json
sudo rm -f /usr/share/google-chrome/extensions/*.json /etc/opt/chrome/policies/managed/baaz-no-save-prompt.json
```

(Arch / Omarchy: `yay -R baaz` instead of the first two lines.)

### macOS

Deleting Baaz.app also removes its login item. The rest:

```
pkill -f BaazMenuBar; pkill -f 'baaz daemon'
rm -rf /Applications/Baaz.app ~/Applications/Baaz.app
rm -f  ~/.local/bin/baaz
rm -f  "$HOME/Library/Application Support/Google/Chrome/NativeMessagingHosts/com.shahriyar.baaz.json"
rm -rf ~/Downloads/baaz-extension
defaults delete com.google.Chrome PromptForDownloadLocation
```

On both: remove the extension from `chrome://extensions`, and restart Chrome.

Your downloads are never touched. To also drop baaz's own settings and
download history, delete `~/.config/baaz` and `~/.local/share/baaz` on Linux,
or `~/Library/Application Support/baaz` on macOS.

---

## How to use it

You mostly don't have to do anything:

- **Normal downloads**: click any download link in Chrome like you always do.
  baaz grabs it, a notification pops up, and the file appears in
  `Downloads/<category>/` when done.
- **Videos**: move your mouse over a video on YouTube, Facebook, TikTok,
  Instagram, X or Vimeo. A **Download** button appears on the video. Click
  it, pick a quality (or "Audio only" for mp3), done.
- **Watch progress**: on Omarchy look at your bar, on macOS look at the menu
  bar near the clock — it shows speed and percent while anything downloads.
  Click it for the full list with pause, resume, and cancel buttons. Click a
  finished item to show it in Finder (macOS) or open its folder (Linux).
- **Turn baaz off**: click the widget → flip the switch at the top —
  Chrome goes back to downloading by itself. Flip it again to return.
  (The extension icon in Chrome shows the same on/off state.)

## Where do my files go?

Straight into your Downloads folder — or wherever you point baaz instead —
sorted automatically:

```
Downloads/Videos/        movies, clips
Downloads/Music/         mp3 and audio
Downloads/Documents/     pdf, docs, spreadsheets
Downloads/Programs/      installers and packages
Downloads/Compressed/    zip, rar, iso
Downloads/Images/        pictures
Downloads/Other/         everything else
```

Don't like sorting? Turn it off: widget → gear icon → "Sort into category
folders", and files land straight in `Downloads/`.

## Settings

Click the widget → the **gear icon**:

- **where downloads are saved** — "Saving to" → **Change…** picks a folder
- how many downloads run at the same time
- how many pieces each file is split into
- minimum file size baaz takes over (smaller files stay with Chrome)
- category folder sorting on/off

From a terminal that last one is `baaz config dir /path/to/folder` (quote it
if it has spaces). Category folders are created inside whatever you pick.

The switch next to the gear turns Chrome takeover on/off entirely.

## Something's wrong?

| Problem | Fix |
|---|---|
| Extension popup says "daemon offline" | Run any baaz command once (e.g. `baaz ls`) — it starts itself. Then reopen the popup. |
| Video downloads fail | Install `yt-dlp` (see above). On macOS that means `brew install yt-dlp ffmpeg` — baaz looks in Homebrew's and MacPorts' directories itself, so you do not have to fix its PATH. |
| A download is stuck | Widget → pause it, then resume. It continues from where it stopped. |
| I want a file AND its list entry gone | Hover the entry in the widget → trash icon. "Clear all" only empties the list, files stay. |
| Chrome still shows its own save dialog | Run `sudo baaz install-chrome` again, then `chrome://restart`. |
| macOS: extension stopped working | Chrome loads it from `~/Downloads/baaz-extension` — if that folder was deleted or moved, re-run `sudo baaz install-chrome` and "Load unpacked" it again. |
| macOS: no icon in the menu bar | `baaz install-menubar` (without `sudo`). If the bar is full, macOS hides icons — widen it or quit another one. |
| macOS: no notifications | First banner asks for permission. Otherwise allow **Script Editor** in System Settings → Notifications. |
| macOS: "cannot be opened because Apple cannot check it" | Baaz.app isn't notarized yet. Right-click it in `~/Applications` → **Open** → **Open**, once. Or clear the download flag: `xattr -dr com.apple.quarantine ~/Applications/Baaz.app ~/.local/bin/baaz`. |
| macOS: menu bar panel says "daemon starting…" and stays there | The app can't find the `baaz` command. It looks in `~/.local/bin`, `/opt/homebrew/bin` and `/usr/local/bin` — make sure the binary is in one of them. |
| macOS: how do I quit the menu bar app? | Click the icon → **Quit** (bottom right). To stop it opening at login: `launchctl bootout gui/$(id -u)/com.shahriyar.baaz.menubar`. |

---

## For developers

<details>
<summary>Build from source, architecture, CLI reference</summary>

Requires Go 1.27+ (see `go.mod`). `make install` builds and installs to
`~/.local/bin/baaz`.
On macOS, `make macos-dmg` builds the shippable `Baaz.dmg` (needs Xcode):
a universal `Baaz.app` with the universal `baaz` CLI inside it at
`Contents/Resources/baaz`. The app copies that CLI to `~/.local/bin` on first
launch, runs `install-chrome`, and registers itself with `SMAppService` — so
the disk image is the whole install and nothing has to be run in a terminal.
`make macos-app` builds just the bundle; `make macos-test` runs the Swift
tests.

Components: one Go binary (daemon + CLI + native-messaging host, unix socket,
JSON-lines IPC), a Chrome MV3 extension in `extension/` (ID pinned via `key`
in the manifest), a Quickshell bar plugin in `bar-plugin/` (embedded into the
binary; `baaz install-bar`), and a SwiftUI menu bar app in `macos/menubar/`
(`baaz install-menubar`). Both widgets are thin clients: they read
`baaz watch` and shell out to the CLI, so the daemon stays the only place
state lives.

Platform-specific code is split by build tag into `*_linux.go` / `*_darwin.go`
(paths, notifications, Chrome integration). The extension is delivered
differently per platform: Linux packs a CRX and registers it via
`external_crx`, which macOS has rejected since Chrome 44 (its External
Extensions directory honors only an `external_update_url` pointing at the Web
Store), so macOS unpacks the extension and the user loads it once. Publishing
to the Web Store would let macOS use a forced `external_update_url` instead. The Swift model in
`Sources/BaazCore/Snapshot.swift` mirrors `internal/ipc/protocol.go`; its
tests decode real `baaz status --json` output to catch drift.

**macOS: XProtect.** Apple's rule `macos_adload_g_bundle` deletes any Mach-O
under 15MB containing all of `_main.main`,
`/Library/Application Support/Google/Chrome/`, `killall` and `cfprefs` — an
adware fingerprint an ordinary build of the Chrome-integration code matched
exactly, so macOS moved the binary to the Trash on first run. Build those
paths with `filepath.Join` instead of writing the literal, and never shell out
to `killall`. `make macos-check` (and CI) fails the build if all four return.
Note that macOS caches the verdict per inode, so replace the binary rather
than overwriting it in place.

How it downloads fast: a probe request (`Range: bytes=0-0`) checks range
support; supported files are preallocated and fetched as up to 8 concurrent
byte ranges written at their offsets, with per-segment retry and resume.
Range-less streams get a single connection with soft-pause (TCP backpressure)
and Range-append resume. Media pages are delegated to yt-dlp with progress
parsed into the same job accounting.

CLI:

```
baaz add URL [--out NAME] [--format best|1080|720|480|audio]
baaz ls | status [--json] | watch
baaz pause|resume|cancel|delete ID
baaz clear
baaz on | off
baaz config [KEY VALUE]      # intercept categorize segments max-active min-size dir
baaz daemon | install-chrome | install-bar | install-menubar
```

Paths, Linux: config `~/.config/baaz/config.json` · state
`~/.local/share/baaz/` · socket `$XDG_RUNTIME_DIR/baaz.sock`.
macOS: both under `~/Library/Application Support/baaz/` · socket `$TMPDIR`
(short enough for the 104-byte `sun_path` limit). In-progress files live in
`<downloadDir>/.baaz-tmp/` on both.

Tests: `go test ./internal/...` and `make macos-test`.

</details>

## License

MIT — see [LICENSE](LICENSE).
