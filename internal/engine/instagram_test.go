package engine

import (
	"encoding/json"
	"testing"
)

func TestIsInstagramPostURL(t *testing.T) {
	tests := []struct {
		url   string
		valid bool
	}{
		{"https://www.instagram.com/p/CvYrSgnsKjv/", true},
		{"https://instagram.com/p/ABC_123/", true},
		{"https://www.instagram.com/reel/DFQe23tOWKz/", false},
		{"https://evil.com/?u=https://www.instagram.com/p/CvYrSgnsKjv/", false},
		{"https://youtube.com/watch?v=123", false},
	}
	for _, tt := range tests {
		if got := IsInstagramPostURL(tt.url); got != tt.valid {
			t.Errorf("IsInstagramPostURL(%q) = %v, want %v", tt.url, got, tt.valid)
		}
	}
}

func TestParseInstagramEmbedCarousel(t *testing.T) {
	contextJSON := `{"gql_data":{"shortcode_media":{` +
		`"shortcode":"ABC","owner":{"username":"user"},` +
		`"edge_media_to_caption":{"edges":[{"node":{"text":"my post"}}]},` +
		`"edge_sidecar_to_children":{"edges":[` +
		`{"node":{"is_video":false,"display_url":"https://cdn/1.jpg"}},` +
		`{"node":{"is_video":true,"video_url":"https://cdn/2.mp4"}}]}}}}`
	quoted, err := json.Marshal(contextJSON)
	if err != nil {
		t.Fatal(err)
	}
	html := []byte(`<script>["PolarisEmbedSimple","init",[],[{"contextJSON":` + string(quoted) + `}]],</script>`)

	post, err := parseInstagramEmbedHTML(html, "ABC", "https://www.instagram.com/p/ABC/")
	if err != nil {
		t.Fatal(err)
	}
	if len(post.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(post.Files))
	}
	if post.Files[0].Name != "instagram_ABC_01.jpg" || post.Files[0].URL != "https://cdn/1.jpg" {
		t.Errorf("unexpected first file: %+v", post.Files[0])
	}
	if post.Files[1].Name != "instagram_ABC_02.mp4" || post.Files[1].URL != "https://cdn/2.mp4" {
		t.Errorf("unexpected second file: %+v", post.Files[1])
	}
	if post.Title != "my post" || post.User != "user" {
		t.Errorf("unexpected metadata: title=%q user=%q", post.Title, post.User)
	}
}

func TestParseInstagramEmbedSinglePhoto(t *testing.T) {
	contextJSON := `{"gql_data":{"shortcode_media":{` +
		`"shortcode":"ABC","is_video":false,"display_url":"https://cdn/1.jpg"}}}`
	quoted, err := json.Marshal(contextJSON)
	if err != nil {
		t.Fatal(err)
	}
	html := []byte(`["x","init",[],[{"contextJSON":` + string(quoted) + `}]],`)

	post, err := parseInstagramEmbedHTML(html, "ABC", "https://www.instagram.com/p/ABC/")
	if err != nil {
		t.Fatal(err)
	}
	if len(post.Files) != 1 || post.Files[0].Name != "instagram_ABC.jpg" {
		t.Fatalf("unexpected files: %+v", post.Files)
	}
}
