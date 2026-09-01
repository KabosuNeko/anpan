package engine

import (
	"context"
	"os"
	"testing"
)

func TestFetchLyricsLive(t *testing.T) {
	ctx := context.Background()
	res, err := FetchLyrics(ctx, "Jenny", "Studio Killers", 217)
	if err != nil {
		t.Logf("LRCLIB network test skipped: %v", err)
		return
	}
	if res.SyncedLyrics == "" && res.PlainLyrics == "" {
		t.Errorf("expected lyrics content")
	}
}

func TestSaveLrcFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "anpan-test-audio-*.mp3")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	lyrics := &LyricsResult{
		SyncedLyrics: "[00:01.00] Test line",
	}
	lrcPath, err := SaveLrcFile(tmpFile.Name(), lyrics)
	if err != nil {
		t.Fatalf("SaveLrcFile failed: %v", err)
	}
	defer os.Remove(lrcPath)

	data, err := os.ReadFile(lrcPath)
	if err != nil || string(data) != "[00:01.00] Test line" {
		t.Errorf("unexpected lrc content: %s", string(data))
	}
}
