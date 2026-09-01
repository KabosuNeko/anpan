package system

import (
	"testing"
)

func isolateTestHome(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")
}

func TestAddToHistoryAndLoad(t *testing.T) {
	isolateTestHome(t)

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
