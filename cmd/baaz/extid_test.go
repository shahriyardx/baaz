package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"
)

// chromeExtensionID derives an extension's ID the way Chrome does: SHA-256
// over the public key's DER, and the first 16 bytes rendered with a-p
// standing for the hex digits.
func chromeExtensionID(derBase64 string) (string, error) {
	der, err := base64.StdEncoding.DecodeString(derBase64)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(der)
	out := make([]byte, 0, 32)
	for _, b := range sum[:16] {
		out = append(out, 'a'+(b>>4), 'a'+(b&0x0F))
	}
	return string(out), nil
}

// The Web Store fixes an extension's ID on first upload and never changes
// it. storeExtID has to keep matching the key in the manifest, or a store
// install would be refused by the native-messaging host — silently, from the
// user's point of view, as "not connected".
func TestStoreExtIDMatchesManifestKey(t *testing.T) {
	raw, err := os.ReadFile("../../extension/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m.Key == "" {
		t.Fatal("extension/manifest.json has no key: the ID would no longer be pinned")
	}
	got, err := chromeExtensionID(m.Key)
	if err != nil {
		t.Fatal(err)
	}
	if got != storeExtID {
		t.Errorf("manifest key yields %q but storeExtID is %q.\n"+
			"The published ID cannot change — if the manifest key was edited, put it back.",
			got, storeExtID)
	}
}

// Every ID the host accepts must be a real 32-character extension ID.
func TestAllowedOriginsCoversBothIDs(t *testing.T) {
	got := allowedOrigins("")
	for _, id := range []string{storeExtID} {
		if len(id) != 32 {
			t.Errorf("%q is not a 32-character extension ID", id)
		}
		if !contains(got, "chrome-extension://"+id+"/") {
			t.Errorf("allowed_origins is missing %s:\n  %s", id, got)
		}
	}
	// An override is added, not substituted, so both routes keep working.
	withExtra := allowedOrigins("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if !contains(withExtra, storeExtID) {
		t.Errorf("--ext-id dropped the store ID from allowed_origins")
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}
