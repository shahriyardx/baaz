<p align="center">
  <img src="docs/images/banner@2x.png" alt="Baaz" width="100%">
</p>

<p align="center">
  <a href="https://github.com/shahriyardx/baaz/releases/latest/download/Baaz.dmg"><img alt="Download for macOS" src="https://img.shields.io/badge/Download-macOS%20·%20Baaz.dmg-2563eb?style=flat-square"></a>
  <a href="https://github.com/shahriyardx/baaz/releases/latest"><img alt="Latest release" src="https://img.shields.io/github/v/release/shahriyardx/baaz?style=flat-square&color=555"></a>
  <a href="https://chromewebstore.google.com/detail/nidklljbjhpljgdeebcpbbnbcijbbcdl"><img alt="Chrome Web Store" src="https://img.shields.io/badge/Chrome-Extension-4285F4?style=flat-square"></a>
  <img alt="macOS 13+" src="https://img.shields.io/badge/macOS-13%2B-555?style=flat-square">
  <a href="LICENSE"><img alt="MIT" src="https://img.shields.io/badge/License-MIT-555?style=flat-square"></a>
</p>

**Baaz takes over your Chrome downloads and finishes them in a fraction of
the time.** It splits each file into pieces and pulls them all at once, so a
download that crawled at one connection's pace now uses the whole line. It
also saves video from YouTube, Facebook, TikTok and more, with one click.

*baaz — বাজ — is Bengali for falcon.*

---

## What you get

- **Much faster downloads.** A large file arrives in up to 8 pieces at once
  instead of one, then gets stitched back together.
- **Nothing is ever downloaded twice.** Pause a file, close the app, reboot,
  come back tomorrow — hit resume and it carries on from the exact point it
  reached. An interrupted download is held, not thrown away.
- **One-click video saving.** Hover any video on YouTube, Facebook, TikTok,
  Instagram, X or Vimeo and a Download button appears. Pick a quality, or
  grab just the audio as an mp3.
- **A real Mac app.** Every download in one window, with search, filters, a
  details panel, and drag-free installing.
- **Always in your menu bar.** Live speed and progress next to the clock,
  and a click away from pausing anything.
- **Tidy folders.** Files sort themselves into Videos, Music, Documents,
  Programs and so on — or don't, if you'd rather they didn't.
- **Updates itself.** Baaz checks daily, tells you when there's a new
  version, and installs it in place.
- **Nothing else to install.** The disk image is the whole thing — the
  video tools arrive in the background the first time you open Baaz.

---

## Install

### macOS

1. Download **[Baaz.dmg][dmg]**.
2. Drag **Baaz** onto **Applications**.
3. Open it. The first time, **right-click the app → Open** — Baaz isn't
   notarized by Apple yet, so a plain double-click is refused once.

A falcon appears in your menu bar, and that's the install finished. Opening
the app connects Chrome and sets Baaz to start with your Mac.

Then install the Chrome extension: **[Baaz on the Chrome Web Store][ext]**,
and restart Chrome.

Works on macOS 13 (Ventura) and newer, on both Intel and Apple Silicon.

[dmg]: https://github.com/shahriyardx/baaz/releases/latest/download/Baaz.dmg
[ext]: https://chromewebstore.google.com/detail/nidklljbjhpljgdeebcpbbnbcijbbcdl

### Linux

**Arch / Omarchy:**

```
yay -S baaz
```

**Anything else** — paste this into a terminal:

```
curl -fsSL https://github.com/shahriyardx/baaz/releases/latest/download/install.sh | bash
```

Run it as yourself, **not** with `sudo`. It asks for your password once, at
the start, and only for the Chrome step.

Then install the extension from the **[Chrome Web Store][ext]**, restart
Chrome, and run this once so Chrome can talk to Baaz:

```
sudo baaz install-chrome
```

For the Omarchy bar widget:

```
baaz install-bar
```

Video downloads on Linux need `yt-dlp` and `ffmpeg`; the installer offers to
get them, or use your package manager. On macOS there is nothing to install:
Baaz fetches both in the background the first time you open it, showing what
it is doing in the menu bar, and uses a copy you already have if it finds
one. Nothing waits on it — downloads work throughout.

---

## Using it

Mostly, you don't.

**Downloading a file** — click any download link in Chrome the way you always
have. Baaz picks it up, shows a notification, and the finished file lands in
your Downloads folder.

**Saving a video** — move your mouse over a video, click the **Download**
button that appears, and choose a size. The menu only lists sizes that video
actually has. Most sizes up to 1080p play in anything; larger ones are
labelled with their video format, because some older players won't open them.
There's an **Audio only** option for mp3.

**Watching progress** — the menu bar shows speed and percent while anything
is downloading. Click it for the list, with pause, resume and cancel. Click a
finished file to reveal it in Finder.

**Turning it off** — click the falcon and flip the switch. Chrome goes back
to downloading on its own; flip it again to hand back over.

---

## The app

Open Baaz from the Dock, or click the falcon and choose **Baaz Downloads**.

- Every download in one list, filtered by **Downloading**, **Paused**,
  **Completed** and **Failed**, with a search box.
- **＋** or <kbd>⌘N</kbd> adds a link by hand. If you've just copied a URL
  it's filled in for you, and video pages get a quality picker.
- Pause, resume or cancel one download, or all of them at once
  (<kbd>⌘.</kbd> and <kbd>⇧⌘R</kbd>).
- Click **parts** on a running download to watch the pieces arrive.
- Click any download for a details panel: size, speed, time left, where it's
  saving, the original link, and when it started and finished.
- Right-click a finished download to reveal it in Finder or delete it.

Closing the window doesn't quit Baaz — downloads keep running and the falcon
stays in the menu bar. Quit from the menu, or <kbd>⌘Q</kbd>.

---

## Where files go

Into your Downloads folder, sorted for you:

```
Downloads/Videos/        films and clips
Downloads/Music/         mp3 and other audio
Downloads/Documents/     pdfs, documents, spreadsheets
Downloads/Programs/      installers and apps
Downloads/Compressed/    zip, rar, disk images
Downloads/Images/        pictures
Downloads/Other/         everything else
```

Prefer them all in one place? Turn off **Sort into category folders** in
Settings, and everything lands straight in `Downloads/`.

---

## Settings

Press <kbd>⌘,</kbd>, or click the gear in the menu bar panel.

**General** — where downloads are saved, whether to sort into folders, and
whether Baaz opens at login.

**Downloads** — a speed limit for everything at once, how many downloads run
together, how many pieces each file is split into, and the size below which
Chrome keeps its own downloads.

**Browser** — turn the Chrome takeover on or off, and re-run the Chrome
connection if it ever stops working.

Changing the speed limit takes effect immediately, including on downloads
already running.

---

## Updating

**macOS updates itself.** Baaz checks once a day, tells you when there's a
newer version, and installs it when you say so — no dragging, no security
prompt. **Baaz → Check for Updates…** does it on demand. Every update is
signed, and one that doesn't match Baaz's signature is refused.

**Linux:**

```
yay -Syu baaz                  # Arch / Omarchy
```

On other distributions, run the install command again — it fetches the
current version.

The Chrome extension updates itself.

---

## Something's wrong?

| What you see | What to do |
|---|---|
| No falcon in the menu bar | Open Baaz from Applications. If your menu bar is full, macOS hides icons — quit something else or widen it. |
| "Baaz cannot be opened because Apple cannot check it" | Right-click Baaz in **Applications** → **Open** → **Open**. Once only. |
| The menu bar panel says "starting…" and stays there | Click the falcon → **Quit**, then open Baaz again. If it persists, reinstall from the disk image. |
| Chrome downloads files itself instead | Check the extension is switched on at `chrome://extensions`. Then **Settings → Browser → Re-run Chrome Setup** on macOS, or `sudo baaz install-chrome` on Linux, and restart Chrome. |
| Video downloads fail | On macOS, Baaz fetches its video tools on the first video download; if that was refused or interrupted, try again, or install them yourself with `brew install yt-dlp ffmpeg` — Baaz uses an existing copy if it finds one. On Linux, install `yt-dlp` and `ffmpeg`. |
| A download seems stuck | Pause it, then resume. It carries on from where it stopped. |
| Downloads are paused after a reboot or a crash | That is deliberate — everything already downloaded is kept. Press resume and they carry on. |
| I want the file *and* the list entry gone | Hover the entry and click the trash icon. **Clear all** only empties the list; files stay. |
| No notifications | The first one asks permission. Otherwise allow them in System Settings → Notifications. |
| Chrome still asks where to save | Re-run the Chrome setup above, then restart Chrome. |

---

## Uninstall

### macOS

Dragging **Baaz.app** to the Trash removes the app and stops it opening at
login. To clear the rest:

```
pkill -f BaazMenuBar; pkill -f 'baaz daemon'
rm -rf /Applications/Baaz.app ~/Applications/Baaz.app
rm -f  ~/.local/bin/baaz
rm -f  "$HOME/Library/Application Support/Google/Chrome/NativeMessagingHosts/com.shahriyar.baaz.json"
defaults delete com.google.Chrome PromptForDownloadLocation
```

### Linux

```
pkill -f 'baaz daemon'
rm -f  ~/.local/bin/baaz
rm -rf ~/.config/omarchy/plugins/shahriyardx.baaz
rm -f  ~/.config/google-chrome/NativeMessagingHosts/com.shahriyar.baaz.json
sudo rm -f /etc/opt/chrome/policies/managed/baaz-no-save-prompt.json
```

On Arch / Omarchy, `yay -R baaz` replaces the first two lines.

On both, remove the extension at `chrome://extensions` and restart Chrome.

**Your downloaded files are never touched.** To also drop Baaz's settings and
history, delete `~/Library/Application Support/baaz` on macOS, or
`~/.config/baaz` and `~/.local/share/baaz` on Linux.

---

## For developers

<details>
<summary>Build from source, architecture, CLI reference</summary>

Requires Go 1.27+ (see `go.mod`). `make install` builds and installs to
`~/.local/bin/baaz`.

On macOS, `make macos-dmg` builds the shippable `Baaz.dmg` (needs Xcode): a
universal `Baaz.app` with the universal `baaz` CLI inside it at
`Contents/Resources/baaz`. The app copies that CLI to `~/.local/bin` on first
launch, runs `install-chrome`, and registers itself with `SMAppService` — so
the disk image is the whole install and nothing has to be run in a terminal.
`make macos-app` builds just the bundle; `make macos-test` runs the Swift
tests.

Components: one Go binary (daemon + CLI + native-messaging host, unix socket,
JSON-lines IPC), a Chrome MV3 extension in `extension/`, a Quickshell bar
plugin in `bar-plugin/` (embedded into the binary; `baaz install-bar`), and a
SwiftUI menu bar app in `macos/menubar/`. Both widgets are thin clients: they
read `baaz watch` and shell out to the CLI, so the daemon stays the only
place state lives.

Platform-specific code is split by build tag into `*_linux.go` /
`*_darwin.go` (paths, notifications, Chrome integration). The Swift model in
`Sources/BaazCore/Snapshot.swift` mirrors `internal/ipc/protocol.go`; its
tests decode real `baaz status --json` output to catch drift.

**The extension** ships from the Chrome Web Store on every platform, under a
fixed ID pinned by the `key` field in `extension/manifest.json`. Loading it
unpacked, and the native-messaging manifest, both agree with that ID. The
Store assigns the ID and rejects a package carrying a `key`, so
`make extension-store` strips it from the upload rather than from the
manifest, which every other route depends on. `extension/PUBLISHING.txt` is
the listing written out, ready to paste. The extension's version in its
manifest is its own and moves at the pace store review allows; the app's
version is the git tag.

**Updates.** Sparkle checks `appcast.xml`, published as an asset of each
release, and verifies the archive's EdDSA signature against `SUPublicEDKey`
in Info.plist. That is what makes updates safe without an Apple Developer ID,
and the update is fetched by the app rather than a browser so it carries no
quarantine flag and launches without a Gatekeeper prompt. CI signs with the
`SPARKLE_PRIVATE_KEY` secret. Without it a release still publishes, but no
appcast is generated and existing installs will not see it.

**The disk image.** `macos/make-dmg.sh` lays the installer window out by
driving Finder on the image it is about to ship, then verifies the result
with `macos/dmg/verify-window.sh`. Doing it on the shipped image is not
optional: Finder stores a window background as an alias, an alias resolves by
file id before it falls back to a path, and a layout prepared on any other
volume therefore applies its window size and icon positions perfectly while
silently resolving the background to the wrong file. `macos/dmg/DS_Store` is
only a fallback for a machine with no Finder session, and loses the
background for that reason. The background itself is generated by
`macos/dmg/render-background.swift`, and the README banner by
`docs/render-banner.swift`.

**macOS: XProtect.** Apple's rule `macos_adload_g_bundle` deletes any Mach-O
under 15MB containing all of `_main.main`,
`/Library/Application Support/Google/Chrome/`, `killall` and `cfprefs` — an
adware fingerprint an ordinary build of the Chrome-integration code matched
exactly, so macOS moved the binary to the Trash on first run. Build those
paths with `filepath.Join` instead of writing the literal, and never shell
out to `killall`. `make macos-check` (and CI) fails the build if all four
return. macOS caches the verdict per inode, so replace the binary rather than
overwriting it in place.

**How it downloads fast.** A probe request (`Range: bytes=0-0`) checks range
support; supported files are preallocated and fetched as up to 8 concurrent
byte ranges written at their offsets, with per-segment retry and resume.
Range-less streams get a single connection with soft-pause (TCP backpressure)
and Range-append resume. Media pages are delegated to yt-dlp with progress
parsed into the same job accounting. A shared token bucket caps total
throughput; because yt-dlp fixes `--limit-rate` at exec, a changed cap reruns
it with `-c` rather than waiting for the next download.

CLI:

```
baaz add URL [--out NAME] [--format best|2160|1440|1080|720|480|audio]
baaz ls | status [--json] | watch
baaz pause|resume|cancel|delete ID
baaz clear
baaz on | off
baaz config [KEY VALUE]      # intercept categorize segments max-active min-size speed-limit dir
baaz daemon | install-chrome | install-bar | install-menubar
```

Paths, Linux: config `~/.config/baaz/config.json` · state
`~/.local/share/baaz/` · socket `$XDG_RUNTIME_DIR/baaz.sock`. macOS: both
under `~/Library/Application Support/baaz/` · socket `$TMPDIR` (short enough
for the 104-byte `sun_path` limit). In-progress files live in
`<downloadDir>/.baaz-tmp/` on both.

Tests: `go test ./...` and `make macos-test`.

</details>

## License

MIT — see [LICENSE](LICENSE).
