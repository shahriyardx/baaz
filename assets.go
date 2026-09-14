// Package assets embeds the Chrome extension source.
//
// The extension is published on the Chrome Web Store and installed from
// there on every platform, so nothing here is packed or side-loaded. The
// embed is what `extension/pack-store.sh` and the release workflow build the
// upload from, and it keeps the extension versioned alongside the daemon it
// talks to.
package assets

import "embed"

//go:embed all:extension
var Extension embed.FS

//go:embed all:bar-plugin
var BarPlugin embed.FS
