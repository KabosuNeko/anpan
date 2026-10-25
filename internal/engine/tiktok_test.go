package engine

import (
	"testing"
)

func TestParseTikTokPhotoPage(t *testing.T) {
	html := `<html><script id="__UNIVERSAL_DATA_FOR_REHYDRATION__" type="application/json">` +
		`{"__DEFAULT_SCOPE__":{"webapp.video-detail":{"itemInfo":{"itemStruct":{` +
		`"id":"123","desc":"my post","author":{"uniqueId":"user"},` +
		`"imagePost":{"images":[` +
		`{"imageURL":{"urlList":["https://cdn/a.jpeg?x","https://cdn/a2"]}},` +
		`{"imageURL":{"urlList":["https://cdn/b.jpeg?x"]}}]}}}}}}` +
		`</script></html>`

	post, err := parseTikTokPhotoPage([]byte(html), "https://www.tiktok.com/@user/video/123")
	if err != nil {
		t.Fatal(err)
	}
	if post == nil || len(post.Files) != 2 {
		t.Fatalf("expected 2 photos, got %+v", post)
	}
	if post.Files[0].Name != "tiktok_user_123_photo_01.jpg" {
		t.Errorf("unexpected first filename: %q", post.Files[0].Name)
	}
	if post.Files[0].URL != "https://cdn/a.jpeg?x" {
		t.Errorf("expected the jpeg CDN URL, got %q", post.Files[0].URL)
	}
	if post.Title != "my post" {
		t.Errorf("unexpected title: %q", post.Title)
	}
}

func TestParseTikTokPhotoPageVideoOnly(t *testing.T) {
	html := `<script id="__UNIVERSAL_DATA_FOR_REHYDRATION__" type="application/json">` +
		`{"__DEFAULT_SCOPE__":{"webapp.video-detail":{"itemInfo":{"itemStruct":{"id":"123"}}}}}` +
		`</script>`

	post, err := parseTikTokPhotoPage([]byte(html), "https://www.tiktok.com/@user/video/123")
	if err != nil {
		t.Fatal(err)
	}
	if post != nil {
		t.Fatalf("expected no photo post, got %+v", post)
	}
}

func TestTikTokFetchURL(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"https://www.tiktok.com/@user/photo/7699082500143861023", "https://www.tiktok.com/@i/video/7699082500143861023"},
		{"https://www.tiktok.com/@user/video/7699082500143861023?lang=en", "https://www.tiktok.com/@i/video/7699082500143861023"},
		{"https://vt.tiktok.com/ZS123/", "https://vt.tiktok.com/ZS123/"},
	}
	for _, tc := range cases {
		if got := tiktokFetchURL(tc.in); got != tc.want {
			t.Errorf("tiktokFetchURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
