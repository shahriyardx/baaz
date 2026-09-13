# baaz

**A fast download manager for Linux.** baaz (বাজ — "falcon" in Bengali) takes
over your Chrome downloads and pulls files up to 8× faster by downloading
several pieces at once. It also downloads videos from YouTube, Facebook,
TikTok and more with one click.

- **Faster downloads** — big files split into up to 8 parts, downloaded together
- **Pause and resume** — even after a reboot or crash, downloads continue where they stopped
- **Video download button** — hover any video on YouTube, Facebook, etc. and click Download; pick a quality or audio-only mp3
- **Tidy folders** — files sort themselves into Videos, Music, Documents, Programs and so on inside `Downloads/baaz/`
- **Desktop notifications** — know when a download starts, finishes, or fails
- **Omarchy bar widget** — live speed and progress right in your bar, no window to open

---

## Install

### Arch Linux / Omarchy

```
yay -S baaz
```

### Any other Linux

Paste this into a terminal:

```
curl -fsSL https://github.com/shahriyardx/baaz/releases/latest/download/install.sh | bash
```

That downloads baaz into your home folder. No compiler, no packages, nothing else needed.

---

## Set up Chrome (one time)

1. In a terminal, run:
   ```
   sudo baaz install-chrome
   ```
   (It asks for your password. This connects Chrome to baaz, installs the
   browser extension, and turns off Chrome's "ask where to save" popup.)
2. Restart Chrome — type `chrome://restart` in the address bar and press Enter.
3. Chrome shows a small "Enable extension" message once — click **Enable**.

That's it. From now on, when you download something in Chrome, baaz takes it.

**Optional — video downloads from YouTube etc.** need one extra program:

```
sudo pacman -S yt-dlp        # Arch / Omarchy
sudo apt install yt-dlp      # Ubuntu / Debian
sudo dnf install yt-dlp      # Fedora
```

**Optional — Omarchy bar widget:**

```
baaz install-bar
```

---

## How to use it

You mostly don't have to do anything:

- **Normal downloads**: click any download link in Chrome like you always do.
  baaz grabs it, a notification pops up, and the file appears in
  `Downloads/baaz/<category>/` when done.
- **Videos**: move your mouse over a video on YouTube, Facebook, TikTok,
  Instagram, X or Vimeo. A **Download** button appears on the video. Click
  it, pick a quality (or "Audio only" for mp3), done.
- **Watch progress**: on Omarchy, look at your bar — it shows speed and
  percent while anything downloads. Click it for the full list with pause,
  resume, and cancel buttons. Click a finished item to open its folder.
- **Turn baaz off**: click the bar widget → flip the switch at the top —
  Chrome goes back to downloading by itself. Flip it again to return.
  (The extension icon in Chrome shows the same on/off state.)

## Where do my files go?

Inside your Downloads folder, sorted automatically:

```
Downloads/baaz/Videos/        movies, clips
Downloads/baaz/Music/         mp3 and audio
Downloads/baaz/Documents/     pdf, docs, spreadsheets
Downloads/baaz/Programs/      installers and packages
Downloads/baaz/Compressed/    zip, rar, iso
Downloads/baaz/Images/        pictures
Downloads/baaz/Other/         everything else
```

Don't like sorting? Turn it off: bar widget → gear icon → "Sort into
category folders", and files land straight in `Downloads/`.

## Settings

Click the bar widget → the **gear icon**:

- how many downloads run at the same time
- how many pieces each file is split into
- minimum file size baaz takes over (smaller files stay with Chrome)
- category folder sorting on/off

The switch next to the gear turns Chrome takeover on/off entirely.

## Something's wrong?

| Problem | Fix |
|---|---|
| Extension popup says "daemon offline" | Run any baaz command once (e.g. `baaz ls`) — it starts itself. Then reopen the popup. |
| Video downloads fail | Install `yt-dlp` (see above). |
| A download is stuck | Bar widget → pause it, then resume. It continues from where it stopped. |
| I want a file AND its list entry gone | Hover the entry in the bar widget → trash icon. "Clear all" only empties the list, files stay. |
| Chrome still shows its own save dialog | Run `sudo baaz install-chrome` again, then `chrome://restart`. |

---

## For developers

<details>
<summary>Build from source, architecture, CLI reference</summary>

Requires Go 1.24+. `make install` builds and installs to `~/.local/bin/baaz`.

Components: one Go binary (daemon + CLI + native-messaging host, unix socket
at `$XDG_RUNTIME_DIR/baaz.sock`, JSON-lines IPC), a Chrome MV3 extension in
`extension/` (ID pinned via `key` in the manifest), and a Quickshell bar
plugin in `bar-plugin/` (embedded into the binary; `baaz install-bar`).

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
baaz daemon | install-chrome | install-bar
```

Config: `~/.config/baaz/config.json` · state: `~/.local/share/baaz/` ·
in-progress files: `<downloadDir>/.baaz-tmp/`. Tests: `go test ./internal/...`.

</details>

## License

MIT — see [LICENSE](LICENSE).
