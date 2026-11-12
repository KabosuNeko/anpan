package core

import (
	"testing"
)

func TestIsKnownSite(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{"https://music.youtube.com/watch?v=123", true},
		{"https://www.youtube.com/watch?v=123", true},
		{"https://soundcloud.com/artist/track", true},
		{"https://tiktok.com/@user/video/123", true},
		{"https://vt.tiktok.com/ZS123/", true},
		{"https://tiktokv.com/share/video/123/", true},
		{"https://bandcamp.com/album/xyz", true},
		{"https://example.com/artist/track", false},
	}

	for _, tc := range cases {
		if got := IsKnownSite(tc.url); got != tc.want {
			t.Errorf("IsKnownSite(%q) = %v, want %v", tc.url, got, tc.want)
		}
	}
}

func TestParseURLInput(t *testing.T) {
	t1 := ParseURLInput("https://youtu.be/dQw4w9WgXcQ 01:20-03:45")
	if t1.CleanURL != "https://youtu.be/dQw4w9WgXcQ" || t1.TimeRange != "01:20-03:45" {
		t.Errorf("ParseURLInput with range failed: %+v", t1)
	}

	t2 := ParseURLInput("https://music.youtube.com/watch?v=abc 45-90")
	if t2.CleanURL != "https://music.youtube.com/watch?v=abc" || t2.TimeRange != "45-90" {
		t.Errorf("ParseURLInput with range failed: %+v", t2)
	}

	t3 := ParseURLInput("https://youtu.be/dQw4w9WgXcQ")
	if t3.CleanURL != "https://youtu.be/dQw4w9WgXcQ" || t3.TimeRange != "" {
		t.Errorf("ParseURLInput without range failed: %+v", t3)
	}
}

func TestIsPlaylistURL(t *testing.T) {
	if !IsPlaylistURL("https://music.youtube.com/playlist?list=PL4fGSI1pDJn5kI81J1fYWK5eZRl1zJ5kM") {
		t.Errorf("Expected true for YouTube playlist")
	}
	if !IsPlaylistURL("https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=PL123") {
		t.Errorf("Expected true for YouTube video in playlist")
	}
	if !IsPlaylistURL("https://soundcloud.com/artist/sets/my-album") {
		t.Errorf("Expected true for SoundCloud set")
	}
	if IsPlaylistURL("https://www.youtube.com/watch?v=dQw4w9WgXcQ") {
		t.Errorf("Expected false for single YouTube video")
	}
}

func TestIsTikTokURL(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{"https://www.tiktok.com/@user/video/123", true},
		{"https://vt.tiktok.com/ZS123/", true},
		{"https://tiktokv.com/share/video/123/", true},
		{"https://evil.com/?u=https://tiktok.com/@user/video/123", false},
		{"https://youtube.com/watch?v=123", false},
	}
	for _, tc := range cases {
		if got := IsTikTokURL(tc.url); got != tc.want {
			t.Errorf("IsTikTokURL(%q) = %v, want %v", tc.url, got, tc.want)
		}
	}
}

func TestStripYouTubeMix(t *testing.T) {
	cases := map[string]string{
		"https://youtu.be/Yo77euCUBsk?list=RDYo77euCUBsk":                              "https://youtu.be/Yo77euCUBsk",
		"https://www.youtube.com/watch?v=Yo77euCUBsk&list=RDYo77euCUBsk&start_radio=1": "https://www.youtube.com/watch?v=Yo77euCUBsk",
		"https://www.youtube.com/watch?v=Yo77euCUBsk&list=PL123&index=4":               "https://www.youtube.com/watch?v=Yo77euCUBsk&list=PL123&index=4",
		"https://music.youtube.com/watch?v=Yo77euCUBsk&list=RDCLAK5uy_abc":             "https://music.youtube.com/watch?v=Yo77euCUBsk&list=RDCLAK5uy_abc",
		"https://www.youtube.com/playlist?list=RDYo77euCUBsk":                          "https://www.youtube.com/playlist?list=RDYo77euCUBsk",
		"https://example.com/watch?v=x&list=RDx":                                       "https://example.com/watch?v=x&list=RDx",
	}
	for in, want := range cases {
		if got := StripYouTubeMix(in); got != want {
			t.Errorf("StripYouTubeMix(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHasVideoID(t *testing.T) {
	if !HasVideoID("https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=PL123") || !HasVideoID("https://youtu.be/dQw4w9WgXcQ?list=PL1") {
		t.Errorf("expected video id")
	}
	if HasVideoID("https://www.youtube.com/playlist?list=PL123") || HasVideoID("https://soundcloud.com/a/sets/b") {
		t.Errorf("expected no video id")
	}
}
