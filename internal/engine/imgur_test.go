package engine

import (
	"testing"
)

func TestIsImgurURL(t *testing.T) {
	tests := []struct {
		url   string
		valid bool
	}{
		{"https://imgur.com/a/7eeLGLM", true},
		{"https://imgur.com/gallery/7eeLGLM", true},
		{"https://i.imgur.com/abcde.jpg", false},
		{"https://youtube.com/watch?v=123", false},
	}

	for _, tt := range tests {
		if got := IsImgurURL(tt.url); got != tt.valid {
			t.Errorf("IsImgurURL(%s) = %v, want %v", tt.url, got, tt.valid)
		}
	}
}
