# dm — segmented download manager for Linux

IDM-style downloader: a Go daemon that downloads files in up to 8 parallel
HTTP Range segments, a Chrome extension that hands browser downloads to it,
and an Omarchy (Quickshell) bar widget with live progress.

## Components

- **`dm` binary** — daemon + CLI in one. The daemon auto-starts on first use.
- **`extension/`** — Chrome MV3 extension. Intercepts downloads via
  `downloads.onDeterminingFilename`; if the daemon is unreachable the browser
  download proceeds untouched.
- **`bar-plugin/`** — Omarchy bar widget streaming `dm watch`. Shows count,
  speed and percent while downloading; click for a panel with progress bars,
  pause/resume/cancel, and recent downloads (click a row to open its folder
  with your default file manager via `xdg-open`).

## Install

```sh
make install           # builds and installs ~/.local/bin/dm
make install-plugin    # installs + enables the bar widget
```

Chrome setup — one command:

```sh
sudo dm install-chrome
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
dm add URL [--out NAME]     queue a download
dm ls                       list downloads
dm pause|resume|cancel ID
dm status [--json]          one-shot snapshot
dm watch                    stream JSON snapshots
dm daemon                   run daemon in foreground
```

## Config

`~/.config/dm/config.json` (all optional):

```json
{ "segments": 8, "downloadDir": "~/Downloads", "maxActive": 3, "minSplitSize": 1048576 }
```

State lives in `~/.local/share/dm/` (job files are chmod 600 — they can hold
cookies). Socket: `$XDG_RUNTIME_DIR/dm.sock`.

## How it downloads fast

A probe request (`Range: bytes=0-0`) checks whether the server honors byte
ranges. If yes, the file is preallocated and split into up to 8 contiguous
ranges downloaded concurrently, each written at its own offset. Per-segment
retries resume from the last written byte; the same offsets make kill-and-
resume safe (verified by checksum tests). Servers without range support get a
single-stream download.
