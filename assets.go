// Package assets embeds what `dm install-chrome` deploys: the extension
// source and the key that fixes its extension ID.
//
// The private key is committed on purpose: it only pins the local extension
// ID, it is not a Chrome Web Store identity. Generate a fresh one before any
// Web Store upload.
package assets

import "embed"

//go:embed all:extension
var Extension embed.FS

//go:embed keys/extension-key.pem
var ExtensionKey []byte
