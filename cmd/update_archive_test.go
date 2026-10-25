package cmd

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestChecksumForAsset(t *testing.T) {
	sums := []byte("abc123  other.tar.gz\n" +
		"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef  anpan-linux-amd64.tar.gz\n")
	got, err := checksumForAsset(sums, "anpan-linux-amd64.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	if got != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Fatalf("unexpected checksum %q", got)
	}
	if _, err := checksumForAsset(sums, "missing.tar.gz"); err == nil {
		t.Fatal("expected an error for a missing asset")
	}
}

func TestExtractTarGzBinary(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	payload := []byte("anpan-binary")
	if err := tw.WriteHeader(&tar.Header{Name: "anpan", Mode: 0o755, Size: int64(len(payload))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	archive := filepath.Join(t.TempDir(), "anpan.tar.gz")
	if err := os.WriteFile(archive, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := extractTarGzBinary(archive)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("extracted %q, want %q", got, payload)
	}
}

func TestExtractZipBinary(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("anpan.exe")
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("anpan-exe")
	if _, err := w.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	archive := filepath.Join(t.TempDir(), "anpan.zip")
	if err := os.WriteFile(archive, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := extractZipBinary(archive)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("extracted %q, want %q", got, payload)
	}
}
