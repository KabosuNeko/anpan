package system

import (
	"os"
	"testing"
)

func TestAddToHistoryAndLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "anpan-history-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	AddToHistory("https://example.com/1")
	AddToHistory("https://example.com/2")
	AddToHistory("https://example.com/1") // re-adding should push to front without duplicate

	h := LoadHistory()
	if len(h) != 2 {
		t.Fatalf("expected 2 items, got %d", len(h))
	}
	if h[0] != "https://example.com/1" {
		t.Errorf("expected latest item at index 0, got %s", h[0])
	}
	if h[1] != "https://example.com/2" {
		t.Errorf("expected older item at index 1, got %s", h[1])
	}
}
