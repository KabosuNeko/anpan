package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestIsWatchCandidate(t *testing.T) {
	cases := []struct {
		filename string
		expected bool
	}{
		{"movie.torrent", true},
		{"release.magnet", true},
		{"links.txt", true},
		{"url.link", true},
		{"ignore.mp4", false},
		{"readme.md", false},
	}

	for _, tc := range cases {
		if got := IsWatchCandidate(tc.filename); got != tc.expected {
			t.Errorf("IsWatchCandidate(%q) = %v, expected %v", tc.filename, got, tc.expected)
		}
	}
}

func TestExtractTargetFromWatchFile(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. .torrent file returns itself
	torrentFile := filepath.Join(tmpDir, "sample.torrent")
	_ = os.WriteFile(torrentFile, []byte("d8:announce..."), 0644)
	res, err := extractTargetFromWatchFile(torrentFile)
	if err != nil || res != torrentFile {
		t.Errorf("expected %s, got %s (err: %v)", torrentFile, res, err)
	}

	// 2. .magnet / .txt file returns first non-comment line
	txtFile := filepath.Join(tmpDir, "queue.txt")
	_ = os.WriteFile(txtFile, []byte("# comments\n\nmagnet:?xt=urn:btih:12345\nhttps://other.com"), 0644)
	resTxt, err := extractTargetFromWatchFile(txtFile)
	if err != nil || resTxt != "magnet:?xt=urn:btih:12345" {
		t.Errorf("expected magnet URI, got %s (err: %v)", resTxt, err)
	}
}

func TestProcessWatchFileQuarantinesBadFile(t *testing.T) {
	dir := t.TempDir()
	processedDir := filepath.Join(dir, ".processed")
	failedDir := filepath.Join(dir, ".failed")
	if err := os.MkdirAll(processedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(failedDir, 0o755); err != nil {
		t.Fatal(err)
	}

	badFile := filepath.Join(dir, "broken.txt")
	if err := os.WriteFile(badFile, []byte("# only comments\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var reported error
	cfg := WatcherConfig{WatchDir: dir, OnError: func(_ string, err error) { reported = err }}
	processWatchFile(context.Background(), cfg, processedDir, failedDir, "broken.txt", badFile)

	if reported == nil {
		t.Fatal("expected an error callback for a file with no target")
	}
	entries, err := os.ReadDir(failedDir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected one quarantined file, got %d (err %v)", len(entries), err)
	}
}
