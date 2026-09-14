# Publishing Baaz to the Chrome Web Store

Everything the listing asks for, with drafts to edit. Anything in
**`[FILL IN]`** needs a decision or a URL only you have.

Console: <https://chrome.google.com/webstore/devconsole> ($5 one-time
developer registration.)

---

## 1. Before you upload

- [ ] Bump `version` in `extension/manifest.json` — CI fails the release if it
      does not match the git tag.
- [ ] `make extension-store` → `build/baaz-extension-store.zip`.
      This strips the `key` field; the store assigns its own ID and rejects a
      package carrying one.
- [ ] Publish the privacy policy somewhere with a stable URL
      (see [PRIVACY.md](../PRIVACY.md)) and note the address below.
- [ ] Take at least one screenshot, **1280×800** or 640×400 PNG/JPEG.

---

## 2. Store listing

| Field | Value |
|---|---|
| **Name** | `Baaz` |
| **Summary** (132 char max) | `Faster downloads in Chrome — files are split into parts, and videos download with one click.` *(92)* |
| **Category** | Workflow & Planning *(or Productivity — both fit)* |
| **Language** | English |
| **Website** | `https://github.com/shahriyardx/baaz` |
| **Support URL** | `https://github.com/shahriyardx/baaz/issues` |

### Description

> Baaz makes Chrome's downloads faster and gives you control over them.
>
> **Faster downloads.** Large files are split into as many as 8 parts and
> fetched at once, rather than through a single connection.
>
> **Pause and resume.** Downloads survive a pause, a reboot, or a crash, and
> pick up where they stopped instead of starting over.
>
> **Videos in one click.** Hover a video on YouTube, Facebook, TikTok,
> Instagram, X or Vimeo and click Download. Pick a quality, or take the audio
> only as an mp3.
>
> **Tidy folders.** Files sort themselves into Videos, Music, Documents,
> Programs and so on.
>
> **Nothing leaves your computer.** No account, no sign-in, no server. Baaz
> talks only to the companion app running on your own machine, and the
> extension sends nothing anywhere else.
>
> **Requires the free Baaz app** for macOS or Linux, which does the actual
> downloading: https://github.com/shahriyardx/baaz
>
> Open source, MIT licensed.

### Screenshots — what to capture

1. The popup open, with a download running *(1280×800)*.
2. The Download button hovering over a YouTube video, quality menu open.
3. The Baaz app window with several downloads and the parts view expanded.

> The listing is rejected if a screenshot shows a browser window whose
> content you do not have rights to. Use your own pages or public,
> licence-free video.

---

## 3. Privacy practices

This is the section that decides how long review takes. Every answer below is
true of the code as it stands — check them again if the extension changes.

### Single purpose

> Baaz hands downloads started in Chrome to the Baaz application on the same
> computer, which downloads them faster and can pause and resume them.

### Permission justifications

Paste each one against its permission.

| Permission | Justification |
|---|---|
| `downloads` | Baaz needs to see a download as Chrome starts it, and cancel Chrome's own copy once the Baaz app has taken it over. Without this the file would download twice. |
| `nativeMessaging` | The Baaz application runs on the user's computer and does the downloading. This permission is how the extension hands it the link. It is the extension's entire purpose. |
| `cookies` | Many downloads are behind a login. The cookies for the download's own URL are read and passed to the Baaz app so the file downloads as the signed-in user, exactly as Chrome would have. Cookies are read only for the URL being downloaded, are never stored, and are never sent anywhere except the app on the same computer. |
| `storage` | Stores one setting: whether the user has the extension switched on. Nothing else is kept. |
| `<all_urls>` (host permission) | A download can start from any site, and the Download button has to be able to appear over a video on any site. The extension reads the page only to find the video's address; it does not read or transmit page content. |

### Remote code

> **No.** Every file the extension runs is inside the package. Nothing is
> fetched or evaluated at runtime.

### Data usage — what to tick

Declare **only** this:

- [x] **Authentication information** — cookies for the URL being downloaded,
      used solely to fetch a file the user is entitled to.
- [x] **Website content** — the address of a video on the current page, used
      solely to download it.

Leave everything else unticked: no personally identifiable information, no
health, financial, location, or personal communications data, and no activity
tracking.

### Required certifications

All three are true and can be ticked:

- [x] I do not sell or transfer user data to third parties, apart from the
      approved use cases.
- [x] I do not use or transfer user data for purposes unrelated to my item's
      single purpose.
- [x] I do not use or transfer user data to determine creditworthiness or for
      lending purposes.

### Privacy policy URL

**`[FILL IN]`** — a public URL for [PRIVACY.md](../PRIVACY.md). The simplest
option is the GitHub blob URL:

```
https://github.com/shahriyardx/baaz/blob/master/PRIVACY.md
```

---

## 4. After the first upload

The store assigns the extension ID. Until these two steps are done, an
installed-from-store copy cannot talk to the app, because the app only
accepts messages from the ID it knows.

- [ ] In the dashboard, copy the item's **public key** into
      `extension/manifest.json` as `"key"`. Unpacked loads and the packed CRX
      then share the published ID.
- [ ] Set `defaultExtID` in `cmd/baaz/main.go` to the assigned ID, so the
      native-messaging manifest's `allowed_origins` matches.
- [ ] Reinstall and confirm the popup shows **connected**.

The key committed in `keys/` only ever pinned the local ID. Generate a fresh
one if the extension is ever self-hosted again.

---

## 5. What to expect

A first submission using `nativeMessaging`, `cookies` and `<all_urls>` is
reviewed by hand. Days is normal; weeks happens. Later updates are usually
much quicker.

If it is rejected, the reason is nearly always a permission justification
that does not explain *why the feature cannot work without it*. Rewrite the
justification in those terms rather than restating what the permission does.
