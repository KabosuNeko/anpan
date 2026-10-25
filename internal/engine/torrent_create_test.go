package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateTorrentSingleFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "anpan-seed-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer os.Remove(tmpFile.Name() + ".torrent")

	content := "This is anpan seed test content 12345"
	_, _ = tmpFile.WriteString(content)
	tmpFile.Close()

	created, err := CreateTorrent(tmpFile.Name(), "", nil)
	if err != nil {
		t.Fatalf("unexpected error creating torrent: %v", err)
	}

	if created.TotalBytes != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), created.TotalBytes)
	}
	if len(created.InfoHash) != 40 {
		t.Errorf("expected 40-char infohash, got %s", created.InfoHash)
	}
	if !strings.HasPrefix(created.Magnet, "magnet:?xt=urn:btih:"+created.InfoHash) {
		t.Errorf("expected valid magnet link, got %s", created.Magnet)
	}

	// Verify .torrent exists
	if _, err := os.Stat(created.TorrentPath); err != nil {
		t.Errorf("torrent file does not exist at %s", created.TorrentPath)
	}
}

func TestCreateTorrentDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "anpan-seed-dir-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	f1 := filepath.Join(tmpDir, "file1.txt")
	f2 := filepath.Join(tmpDir, "sub", "file2.txt")
	_ = os.MkdirAll(filepath.Dir(f2), 0755)
	_ = os.WriteFile(f1, []byte("file 1 content"), 0644)
	_ = os.WriteFile(f2, []byte("file 2 content"), 0644)

	outTorrent := filepath.Join(tmpDir, "custom.torrent")
	created, err := CreateTorrent(tmpDir, outTorrent, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedSize := int64(len("file 1 content") + len("file 2 content"))
	if created.TotalBytes != expectedSize {
		t.Errorf("expected %d total bytes, got %d", expectedSize, created.TotalBytes)
	}
	if created.TorrentPath != outTorrent {
		t.Errorf("expected outTorrent %s, got %s", outTorrent, created.TorrentPath)
	}
}

func TestCreateTorrentSkipsHiddenFiles(t *testing.T) {
	srcDir := t.TempDir()
	visible := []byte("visible content")
	if err := os.WriteFile(filepath.Join(srcDir, "visible.txt"), visible, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, ".env"), []byte("SECRET=1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(srcDir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, ".git", "config"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	created, err := CreateTorrent(srcDir, filepath.Join(t.TempDir(), "out.torrent"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.TotalBytes != int64(len(visible)) {
		t.Fatalf("hidden files leaked into torrent: expected %d bytes, got %d", len(visible), created.TotalBytes)
	}
}
