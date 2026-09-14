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
- **A real app on macOS** — a window with every download, add links by hand,
  pause and resume anything, watch the parts arrive
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
connects Chrome and sets itself to open at login.

> On first open, **right-click the app → Open**. baaz isn't notarized yet, so
> a plain double-click is refused once.

macOS 13 (Ventura) or newer, Intel or Apple Silicon. The app will point you
at the Chrome extension, which installs from the Web Store in one click.

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

The installer is Linux-only; on macOS it points you at the disk image and
stops, since the app there carries the CLI and does its own setup.

---

## Set up Chrome (one time)

Install the extension: **[Baaz on the Chrome Web Store][ext]**, then restart
Chrome. That is the whole browser side on every platform.

[ext]: https://chromewebstore.google.com/detail/nidklljbjhpljgdeebcpbbnbcijbbcdl

On **macOS** the app connects Chrome to Baaz by itself when you first open
it. On **Linux**, run this once:

```
sudo baaz install-chrome
```

It writes the native-messaging manifest that lets the extension reach the
daemon, and turns off Chrome's "ask where to save" dialog.

That's it. From now on, when you download something in Chrome, Baaz takes it.

**Video downloads from YouTube etc.** need yt-dlp and ffmpeg. On **macOS**
baaz fetches both itself the first time you download a video — nothing to
install, the first one just takes a little longer while it does. On Linux,
use your package manager (the installer offers to, and a video download that
finds them missing says exactly what to run):

```
sudo pacman -S yt-dlp ffmpeg        # Arch / Omarchy
sudo apt install yt-dlp ffmpeg      # Ubuntu / Debian
sudo dnf install yt-dlp ffmpeg      # Fedora
brew install yt-dlp ffmpeg          # macOS
```

**Optional — the Omarchy bar widget (Linux):**

```
baaz install-bar
```

---

## Updating

**macOS** updates itself. baaz checks once a day and offers the update; you
click Install and it downloads, replaces itself and relaunches — no dragging,
no Gatekeeper prompt. **baaz → Check for Updates…** does it on demand.

Updates are signed with an EdDSA key and refused if the signature does not
match the one built into the app, so a tampered build cannot install itself.

```
yay -Syu baaz            # Arch / Omarchy
```

Other Linux and macOS: run the same curl line from Install again — it fetches
the latest version.

The extension updates itself from the Chrome Web Store, and on macOS the app
updates itself. Nothing to do.

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
defaults delete com.google.Chrome PromptForDownloadLocation
```

On both: remove the extension at `chrome://extensions`, and restart Chrome.

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

## The app (macOS)

Open Baaz from the Dock — or click the falcon and choose **Open baaz** — for
the full window:

- every download in one list, filtered by **Downloading / Paused / Completed /
  Failed**, with a search box
- **＋** or <kbd>⌘N</kbd> to add a link. The URL is filled in from your
  clipboard if you just copied one, and video pages get a quality picker
- pause, resume and cancel per download, or **Pause All** / **Resume All**
  (<kbd>⌘.</kbd> and <kbd>⇧⌘R</kbd>)
- click **parts** on a running download to watch the file arrive in up to 8
  pieces at once
- click a download for a **details panel** on the right: size, speed, time
  left, where it saved, the source link, when it started and finished, and
  the live parts — with Open / Show in Finder / pause / cancel
- right-click a finished one to show it in Finder or delete it

**Settings** live under <kbd>⌘,</kbd> (baaz → Settings…), in three tabs:

- **General** — where downloads are saved, category folders, open at login
- **Downloads** — speed limit, how many at once, parts per file, the size
  below which Chrome keeps its own downloads
- **Browser** — Chrome takeover on/off, and the extension folder with a
  button to re-run the Chrome setup

The gear in the menu bar panel opens the same window.

Closing the window doesn't quit — downloads keep running and the falcon stays
in the menu bar. Quit from the menu or <kbd>⌘Q</kbd>.

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

Don't like sorting? Turn it off in Settings → General → "Sort into category
folders", and files land straight in `Downloads/`.

## Settings

Settings (<kbd>⌘,</kbd>, or the gear in the menu bar panel):

- **where downloads are saved** — "Saving to" → **Change…** picks a folder
- **speed limit** — cap total throughput so a download stops saturating your line
- how many downloads run at the same time
- how many pieces each file is split into
- minimum file size baaz takes over (smaller files stay with Chrome)
- category folder sorting on/off

Category folders are created inside whatever you pick. (On Linux the same
setting is `baaz config dir /path/to/folder`.)

The switch next to the gear turns Chrome takeover on/off entirely.

## Something's wrong?

| Problem | Fix |
|---|---|
| Extension popup says "daemon offline" | Run any baaz command once (e.g. `baaz ls`) — it starts itself. Then reopen the popup. |
| Video downloads fail | Linux: install `yt-dlp` and `ffmpeg`. macOS: baaz fetches them itself, so this usually means it could not reach GitHub — check the network and retry the download. An existing Homebrew or MacPorts copy is used as-is. |
| A download is stuck | Widget → pause it, then resume. It continues from where it stopped. |
| I want a file AND its list entry gone | Hover the entry in the widget → trash icon. "Clear all" only empties the list, files stay. |
| Chrome still shows its own save dialog | Run `sudo baaz install-chrome` again, then `chrome://restart`. |
| Extension stopped working | Check it is enabled at `chrome://extensions`. If downloads still go to Chrome, use **Settings → Browser → Re-run Chrome Setup** (macOS) or `sudo baaz install-chrome` (Linux), then restart Chrome. |
| macOS: no icon in the menu bar | Open Baaz from Applications. If the bar is full, macOS hides icons — widen it or quit another one. |
| macOS: no notifications | First banner asks for permission. Otherwise allow **Script Editor** in System Settings → Notifications. |
| macOS: "cannot be opened because Apple cannot check it" | Baaz.app isn't notarized yet. Right-click it in **Applications** → **Open** → **Open**, once. |
| macOS: panel says "daemon starting…" and stays there | Quit Baaz (falcon → Quit) and open it again. If it persists, reinstall from the disk image. |
| macOS: how do I quit? | Click the falcon → **Quit**, or ⌘Q. To stop it opening at login: **Settings → General → Open baaz at login**. |

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

**Updates.** Sparkle checks `appcast.xml`, published as an asset of each
release, and verifies the archive's EdDSA signature against `SUPublicEDKey`
in Info.plist. That is what makes updates safe without an Apple Developer ID,
and the update is fetched by the app rather than a browser so it carries no
quarantine flag and launches without a Gatekeeper prompt.

CI signs the archive with the `SPARKLE_PRIVATE_KEY` secret — the private half
of that key pair, exported from the maintainer's keychain with
`generate_keys -x`. Without the secret a release still publishes, but no
appcast is generated and existing installs will not see it.

**Two version numbers.** The extension's version in
`extension/manifest.json` is its own: it belongs to the Chrome Web Store
listing and moves when the extension changes, at the pace store review
allows. The app's version is the git tag. They used to be forced equal,
which meant every app release demanded a store re-upload and a fresh review
for an extension that had not changed.

**Publishing the extension.** `extension/PUBLISHING.txt` is the Web Store
listing written out, ready to paste. `make extension-store` builds the upload. The
only difference from what baaz ships is the `key` field: locally it pins the
extension ID so unpacked loads, the packed CRX and the native-messaging
manifest all agree, but the store ignores it, assigns an ID of its own, and
rejects a package that carries one — so the script strips it rather than
removing it from the manifest, which would break every other install route.

After the first upload, copy the public key the dashboard shows into
`extension/manifest.json` as `key`, and set `defaultExtID` to the ID the
dashboard assigns. Every route then shares the published ID. The key
currently committed in `keys/` only pins the local ID; generate a fresh one
if the extension is ever self-hosted again.

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
baaz config [KEY VALUE]      # intercept categorize segments max-active min-size speed-limit dir
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
