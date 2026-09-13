package crx

import (
	"archive/zip"
	"bytes"
	"io/fs"
	"os"
	"testing"
	"testing/fstest"
)

func TestPackIDAndZip(t *testing.T) {
	keyPEM, err := os.ReadFile("../../keys/extension-key.pem")
	if err != nil {
		t.Skip("extension key not present")
	}
	src := fstest.MapFS{
		"manifest.json": {Data: []byte(`{"name":"x"}`)},
		"background.js": {Data: []byte(`// bg`)},
	}
	data, id, err := Pack(src, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	const want = "bekhpkepdgjmplfdclkflkkhbpbgeihl"
	if id != want {
		t.Fatalf("id = %s, want %s", id, want)
	}
	if !bytes.HasPrefix(data, []byte("Cr24")) {
		t.Fatal("missing Cr24 magic")
	}
	// zip payload must be readable and complete
	idx := bytes.Index(data, []byte("PK\x03\x04"))
	if idx < 0 {
		t.Fatal("no zip payload")
	}
	zr, err := zip.NewReader(bytes.NewReader(data[idx:]), int64(len(data)-idx))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, f := range zr.File {
		got[f.Name] = true
	}
	for name := range src {
		if !got[name] {
			t.Fatalf("zip missing %s", name)
		}
	}
	var _ fs.FS = src
}
