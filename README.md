# baaz — segmented download manager for Linux

IDM-style downloader: a Go daemon that downloads files in up to 8 parallel
HTTP Range segments, a Chrome extension that hands browser downloads to it,
and an Omarchy (Quickshell) bar widget with live progress.

## Components

- **`baaz` binary** — daemon + CLI in one. The daemon auto-starts on first use.
- **`extension/`** — Chrome MV3 extension. Intercepts downloads via
  `downloads.onDeterminingFilename`; if the daemon is unreachable the browser
  download proceeds untouched.
- **`bar-plugin/`** — Omarchy bar widget streaming `baaz watch`. Shows count,
  speed and percent while downloading; click for a panel with progress bars,
  pause/resume/cancel, and recent downloads (click a row to open its folder
  with your default file manager via `xdg-open`).

## Install

```sh
make install           # builds and installs ~/.local/bin/baaz
baaz install-bar       # installs + enables the Omarchy bar widget (embedded in the binary)
```

Chrome setup — one command:

```sh
sudo baaz install-chrome
```

It installs everything: native-messaging manifests (for Chrome and
Chromium), a managed policy that disables the "ask where to save" dialog,
and the extension itself (packed to CRX in-process and registered as an
external extension under `/usr/share/*/extensions`). Restart Chrome and
confirm the one-time "Enable extension" prompt.

Without sudo it still writes the user-level native-messaging manifests and
prints what was skipped. The extension ID is pinned by `keys/extension-key.pem`
(committed on purpose — it is a local ID pin, not a Web Store identity;
generate a fresh key before any Web Store upload).

## CLI

```
baaz add URL [--out NAME]     queue a download
baaz ls                       list downloads
baaz pause|resume|cancel ID
baaz status [--json]          one-shot snapshot
baaz watch                    stream JSON snapshots
baaz daemon                   run daemon in foreground
```

## Config

`~/.config/baaz/config.json` (all optional):

```json
{ "segments": 8, "downloadDir": "~/Downloads", "maxActive": 3, "minSplitSize": 1048576 }
```

State lives in `~/.local/share/baaz/` (job files are chmod 600 — they can hold
cookies). Socket: `$XDG_RUNTIME_DIR/baaz.sock`.

## How it downloads fast

A probe request (`Range: bytes=0-0`) checks whether the server honors byte
ranges. If yes, the file is preallocated and split into up to 8 contiguous
ranges downloaded concurrently, each written at its own offset. Per-segment
retries resume from the last written byte; the same offsets make kill-and-
resume safe (verified by checksum tests). Servers without range support get a
single-stream download.
