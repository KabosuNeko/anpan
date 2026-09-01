package engine

import (
	"context"
	"testing"
)

func TestIsBooruURL(t *testing.T) {
	tests := []struct {
		url   string
		valid bool
	}{
		{"https://yande.re/post/show/123456", true},
		{"https://konachan.com/post/show/789012", true},
		{"https://safebooru.org/index.php?page=post&s=view&id=345678", true},
		{"https://gelbooru.com/index.php?page=post&s=view&id=901234", true},
		{"https://youtube.com/watch?v=123", false},
	}

	for _, tt := range tests {
		if got := IsBooruURL(tt.url); got != tt.valid {
			t.Errorf("IsBooruURL(%s) = %v, want %v", tt.url, got, tt.valid)
		}
	}
}

func TestProbeBooruLive(t *testing.T) {
	ctx := context.Background()
	post, err := ProbeBooruPost(ctx, "https://yande.re/post/show/1268141")
	if err != nil {
		t.Logf("Booru live probe failed (network): %v", err)
		return
	}
	if len(post.Files) == 0 {
		t.Errorf("expected booru files, got 0")
	}
	if post.Service != "yandere" {
		t.Errorf("expected service yandere, got %s", post.Service)
	}
}
