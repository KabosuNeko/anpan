package cmd

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

const maxBinarySize = 200 << 20

func checksumForAsset(sums []byte, asset string) (string, error) {
	for _, line := range strings.Split(string(sums), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if strings.TrimPrefix(fields[1], "*") == asset {
			expected := strings.ToLower(fields[0])
			if len(expected) != sha256.Size*2 {
				return "", fmt.Errorf("invalid checksum for %s", asset)
			}
			return expected, nil
		}
	}
	return "", fmt.Errorf("checksum for %s not found", asset)
}

func sha256File(name string) (string, error) {
	f, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func verifySHA256(archivePath, sumsURL, asset, userAgent string) error {
	req, err := http.NewRequest("GET", sumsURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download checksums failed (HTTP %d)", resp.StatusCode)
	}

	sums, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	expected, err := checksumForAsset(sums, asset)
	if err != nil {
		return err
	}

	actual, err := sha256File(archivePath)
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("checksum mismatch for %s", asset)
	}
	return nil
}

func extractBinary(archivePath string) ([]byte, error) {
	if strings.HasSuffix(strings.ToLower(archivePath), ".zip") {
		return extractZipBinary(archivePath)
	}
	return extractTarGzBinary(archivePath)
}

func extractTarGzBinary(archivePath string) ([]byte, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if hdr.Typeflag != tar.TypeReg || path.Base(hdr.Name) != "anpan" {
			continue
		}
		return io.ReadAll(io.LimitReader(tr, maxBinarySize))
	}
	return nil, fmt.Errorf("anpan binary not found in archive")
}

func extractZipBinary(archivePath string) ([]byte, error) {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	for _, f := range zr.File {
		if path.Base(f.Name) != "anpan.exe" && path.Base(f.Name) != "anpan" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(io.LimitReader(rc, maxBinarySize))
		rc.Close()
		if err != nil {
			return nil, err
		}
		return data, nil
	}
	return nil, fmt.Errorf("anpan binary not found in archive")
}
