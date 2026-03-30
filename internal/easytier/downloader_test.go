package easytier

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestFileLooksLikeZip(t *testing.T) {
	dir := t.TempDir()
	plain := filepath.Join(dir, "raw.bin")
	if err := os.WriteFile(plain, []byte{0x4d, 0x5a}, 0o644); err != nil {
		t.Fatal(err)
	}
	if fileLooksLikeZip(plain) {
		t.Fatal("PE magic should not look like zip")
	}

	zpath := filepath.Join(dir, "a.zip")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("easytier-core.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(zpath, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if !fileLooksLikeZip(zpath) {
		t.Fatal("zip magic should be detected")
	}
}

func TestAssetURLPathLooksLikeZip_QueryString(t *testing.T) {
	u := "https://release-assets.githubusercontent.com/foo/easytier-windows-amd64.zip?token=abc"
	if !assetURLPathLooksLikeZip(u) {
		t.Fatal("path suffix .zip should be detected despite query string")
	}
}
