package engine

import (
	"context"
	"testing"
)

func TestIsPixivURL(t *testing.T) {
	tests := []struct {
		url   string
		valid bool
		id    string
	}{
		{"https://www.pixiv.net/artworks/114999991", true, "114999991"},
		{"https://pixiv.net/en/artworks/123456", true, "123456"},
		{"https://pixiv.net/i/789012", true, "789012"},
		{"https://youtube.com/watch?v=123", false, ""},
	}

	for _, tt := range tests {
		if got := IsPixivURL(tt.url); got != tt.valid {
			t.Errorf("IsPixivURL(%s) = %v, want %v", tt.url, got, tt.valid)
		}
		if tt.valid {
			if id := ParsePixivID(tt.url); id != tt.id {
				t.Errorf("ParsePixivID(%s) = %v, want %v", tt.url, id, tt.id)
			}
		}
	}
}

func TestProbePixivPostLive(t *testing.T) {
	ctx := context.Background()
	post, err := ProbePixivPost(ctx, "https://www.pixiv.net/artworks/114999991")
	if err != nil {
		t.Logf("Pixiv live probe failed (network): %v", err)
		return
	}
	if len(post.Files) == 0 {
		t.Errorf("expected pixiv files, got 0")
	}
	if post.Service != "pixiv" {
		t.Errorf("expected service pixiv, got %s", post.Service)
	}
}
