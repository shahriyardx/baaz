# Baaz — Privacy Policy

_Last updated: 2026-09-15_

Baaz is a download manager: a Chrome extension and an application that runs on
your own computer. This policy covers both.

## The short version

Baaz has no servers. It collects nothing, sends nothing, and has no account to
sign in to. Everything it touches stays on your computer.

## What Baaz handles, and why

**Download addresses (URLs).** When you start a download, the extension passes
its address to the Baaz application on the same computer so it can fetch the
file. Video pages additionally have the video's address read from the page,
for the same reason.

**Cookies.** Many files are behind a login. For the address being downloaded —
and only that address — Baaz reads the cookies Chrome would itself have sent,
and passes them to the application on the same computer so the file downloads
as the signed-in you. They are not stored and are not written to disk.

**One setting.** Whether you have the extension switched on, kept in Chrome's
local storage.

**Your downloads and their history.** The application records what you have
downloaded so it can show the list and resume interrupted transfers. This sits
in a file in your own user folder and is never uploaded.

## What Baaz does not do

- No analytics, telemetry, crash reporting or tracking of any kind.
- No advertising, and no profiles built about you.
- Nothing is sold, shared or transferred to anyone.
- No account, sign-in or cloud service exists.
- No data leaves your computer. The extension makes no network requests of its
  own; it talks only to the Baaz application over Chrome's native messaging,
  which cannot reach beyond the machine it runs on.

## Downloads you ask for

Baaz connects to the servers hosting the files you choose to download, exactly
as Chrome would. Those servers see the request as your browser would have made
it. Baaz sends them nothing extra.

For videos, Baaz uses **yt-dlp** and **ffmpeg**, downloaded from their official
release pages the first time they are needed. They run on your computer and are
governed by their own licences.

## Removing your data

Uninstall the extension in Chrome, and delete the application's folder:

- macOS: `~/Library/Application Support/baaz`
- Linux: `~/.config/baaz` and `~/.local/share/baaz`

Downloaded files are yours and are left alone.

## Changes

Any change to this policy will appear in this file, with a new date above.

## Contact

Questions or concerns: <https://github.com/shahriyardx/baaz/issues>
