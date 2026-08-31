package engine

import (
	"testing"
)

func TestIsArchivePostURL(t *testing.T) {
	validUrls := []string{
		"https://kemono.cr/patreon/user/90822862/post/147648418",
		"https://kemono.su/fanbox/user/3873554/post/12509033",
		"https://kemono.party/subscribestar/user/123/post/456",
		"https://coomer.su/onlyfans/user/model1/post/9999",
		"https://coomer.st/fansly/user/model2/post/8888",
		"https://pawchive.st/fanbox/user/3873554/post/12509033",
		"https://pawchive.pw/patreon/user/555/post/777",
		"http://pawchive.st/fantia/user/999/post/111",
	}

	for _, u := range validUrls {
		if !IsArchivePostURL(u) {
			t.Errorf("IsArchivePostURL(%q) = false, want true", u)
		}
	}

	invalidUrls := []string{
		"https://youtube.com/watch?v=dQw4w9WgXcQ",
		"https://kemono.cr/artists",
		"https://kemono.cr/patreon/user/90822862",
		"https://pawchive.st/",
		"https://pawchive.pw/fanbox/user/3873554",
		"https://example.com/file.zip",
	}

	for _, u := range invalidUrls {
		if IsArchivePostURL(u) {
			t.Errorf("IsArchivePostURL(%q) = true, want false", u)
		}
	}
}

func TestParseArchiveURL(t *testing.T) {
	k := ParseArchiveURL("https://kemono.cr/patreon/user/90822862/post/147648418")
	if k == nil || k.Domain != "kemono.cr" || k.Service != "patreon" || k.User != "90822862" || k.ID != "147648418" {
		t.Errorf("ParseArchiveURL failed: %+v", k)
	}

	p := ParseArchiveURL("https://pawchive.pw/fanbox/user/3873554/post/12509033")
	if p == nil || p.Domain != "pawchive.pw" || p.Service != "fanbox" || p.User != "3873554" || p.ID != "12509033" {
		t.Errorf("ParseArchiveURL failed: %+v", p)
	}

	c := ParseArchiveURL("https://coomer.st/onlyfans/user/model_abc/post/123456")
	if c == nil || c.Domain != "coomer.st" || c.Service != "onlyfans" || c.User != "model_abc" || c.ID != "123456" {
		t.Errorf("ParseArchiveURL failed: %+v", c)
	}
}
