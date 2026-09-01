package engine

import (
	"context"
	"testing"
)

func TestIsCloudHostURL(t *testing.T) {
	tests := []struct {
		url   string
		valid bool
	}{
		{"https://pixeldrain.com/u/abc12345", true},
		{"https://drive.google.com/file/d/1A2B3C4D5E/view", true},
		{"https://drive.google.com/open?id=1A2B3C4D5E", true},
		{"https://files.catbox.moe/abc.mp4", true},
		{"https://litterbox.catbox.moe/xyz.zip", true},
		{"https://www.mediafire.com/file/o8t0c3k8n9u503o/sample.txt/file", true},
		{"https://youtube.com/watch?v=123", false},
	}

	for _, tt := range tests {
		if got := IsCloudHostURL(tt.url); got != tt.valid {
			t.Errorf("IsCloudHostURL(%s) = %v, want %v", tt.url, got, tt.valid)
		}
	}
}

func TestProbeCloudHostCatbox(t *testing.T) {
	ctx := context.Background()
	file, err := ProbeCloudHost(ctx, "https://files.catbox.moe/test.mp4")
	if err != nil {
		t.Fatalf("ProbeCloudHost failed: %v", err)
	}
	if file.Filename != "test.mp4" {
		t.Errorf("expected filename test.mp4, got %s", file.Filename)
	}
}

func TestIsPixeldrainListURL(t *testing.T) {
	if !IsPixeldrainListURL("https://pixeldrain.com/l/abc12345") {
		t.Errorf("expected valid pixeldrain list url")
	}
	if IsPixeldrainListURL("https://pixeldrain.com/u/abc12345") {
		t.Errorf("expected false for /u/ url")
	}
}
