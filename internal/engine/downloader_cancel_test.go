//go:build !windows

package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBakeVideoCancelInterruptsYtDlp(t *testing.T) {
	dir := t.TempDir()
	mark := filepath.Join(dir, "interrupted")
	fake := filepath.Join(dir, "yt-dlp")
	script := "#!/bin/sh\ntrap 'echo int > \"$ANPAN_TEST_MARK\"; exit 1' INT\nwhile :; do sleep 0.1; done\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ANPAN_TEST_MARK", mark)

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(300*time.Millisecond, cancel)
	_, err := BakeVideo(ctx, BakeVideoOptions{YtdlpBin: fake, URL: "https://example.com/v", OutputDir: dir}, BakeHandlers{})
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if _, statErr := os.Stat(mark); statErr != nil {
		t.Errorf("yt-dlp should receive SIGINT on cancel so it can stop its children")
	}
}
