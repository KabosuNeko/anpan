package engine

import (
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
