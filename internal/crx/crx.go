// Package crx packs an extension directory into a CRX3 file — the signed
// format Chrome accepts for external (non-Web-Store) installs.
package crx

import (
	"archive/zip"
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"io/fs"
	"sort"
)

// Pack zips src and wraps it in a CRX3 container signed with the RSA key in
// keyPEM. Returns the crx bytes and the extension ID the key produces.
func Pack(src fs.FS, keyPEM []byte) ([]byte, string, error) {
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, "", fmt.Errorf("no PEM block in key")
	}
	key, err := parseRSAKey(block.Bytes)
	if err != nil {
		return nil, "", err
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, "", err
	}

	archive, err := zipFS(src)
	if err != nil {
		return nil, "", err
	}

	digest := sha256.Sum256(pubDER)
	crxID := digest[:16]

	signedData := protoBytes(1, crxID) // SignedData{ crx_id }

	// Signature covers: "CRX3 SignedData\x00" + le32(len) + signedData + zip
	h := sha256.New()
	h.Write([]byte("CRX3 SignedData\x00"))
	binary.Write(h, binary.LittleEndian, uint32(len(signedData)))
	h.Write(signedData)
	h.Write(archive)
	sig, err := rsa.SignPKCS1v15(nil, key, crypto.SHA256, h.Sum(nil))
	if err != nil {
		return nil, "", err
	}

	// CrxFileHeader{ sha256_with_rsa(2) = AsymmetricKeyProof{pub(1), sig(2)},
	//                signed_header_data(10000) }
	proof := append(protoBytes(1, pubDER), protoBytes(2, sig)...)
	header := append(protoBytes(2, proof), protoBytes(10000, signedData)...)

	var out bytes.Buffer
	out.WriteString("Cr24")
	binary.Write(&out, binary.LittleEndian, uint32(3))
	binary.Write(&out, binary.LittleEndian, uint32(len(header)))
	out.Write(header)
	out.Write(archive)
	return out.Bytes(), idString(crxID), nil
}

func parseRSAKey(der []byte) (*rsa.PrivateKey, error) {
	if k, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return k, nil
	}
	k, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, err
	}
	rk, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is not RSA")
	}
	return rk, nil
}

// idString maps the 16-byte crx id to Chrome's a-p alphabet.
func idString(id []byte) string {
	out := make([]byte, 0, 32)
	for _, b := range id {
		out = append(out, 'a'+b>>4, 'a'+b&0xf)
	}
	return string(out)
}

// protoBytes encodes one length-delimited protobuf field.
func protoBytes(field uint64, data []byte) []byte {
	buf := make([]byte, 0, len(data)+12)
	buf = binary.AppendUvarint(buf, field<<3|2)
	buf = binary.AppendUvarint(buf, uint64(len(data)))
	return append(buf, data...)
}

func zipFS(src fs.FS) ([]byte, error) {
	var names []string
	err := fs.WalkDir(src, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			names = append(names, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(names)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range names {
		data, err := fs.ReadFile(src, name)
		if err != nil {
			return nil, err
		}
		w, err := zw.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(data); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
