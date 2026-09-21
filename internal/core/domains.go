package core

import (
	"net/url"
	"regexp"
	"strings"
)

var knownHosts = []string{
	"youtube.com", "youtu.be", "music.youtube.com",
	"x.com", "twitter.com",
	"instagram.com",
	"threads.net", "threads.com",
	"tiktok.com",
	"vimeo.com",
	"twitch.tv",
	"reddit.com",
	"facebook.com", "fb.watch",
	"soundcloud.com",
	"bandcamp.com",
	"kemono.cr", "kemono.su", "kemono.party",
	"coomer.su", "coomer.party", "coomer.st",
	"pawchive.st", "pawchive.pw",
	"pixiv.net", "pixiv.me",
	"yande.re",
	"konachan.com", "konachan.net",
	"safebooru.org",
	"gelbooru.com",
	"pixeldrain.com",
	"drive.google.com",
	"catbox.moe",
	"mediafire.com",
	"imgur.com",
	"archive.org",
}

// IsKnownSite reports whether rawURL's host matches a known media site, including its subdomains.
func IsKnownSite(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	hostname := strings.ToLower(u.Hostname())

	for _, h := range knownHosts {
		if hostname == h || strings.HasSuffix(hostname, "."+h) {
			return true
		}
	}
	return false
}

var timeRangeRegex = regexp.MustCompile(`(?i)(?:^|\s+)((?:\d{1,2}:)?\d{1,2}:\d{2}|\d+)\s*-\s*((?:\d{1,2}:)?\d{1,2}:\d{2}|\d+)\s*$`)

type ParsedInput struct {
	CleanURL  string
	TimeRange string
	TimeLabel string
}

func ParseURLInput(input string) ParsedInput {
	trimmed := strings.TrimSpace(input)
	loc := timeRangeRegex.FindStringSubmatchIndex(trimmed)
	if loc == nil {
		return ParsedInput{CleanURL: trimmed}
	}

	match := timeRangeRegex.FindStringSubmatch(trimmed)
	start := match[1]
	end := match[2]
	cleanURL := strings.TrimSpace(trimmed[:loc[0]])

	return ParsedInput{
		CleanURL:  cleanURL,
		TimeRange: start + "-" + end,
		TimeLabel: start + " → " + end,
	}
}

func IsLikelyTarget(input string) bool {
	trimmed := strings.TrimSpace(input)
	if strings.HasPrefix(trimmed, "magnet:?") {
		return true
	}
	if strings.HasSuffix(trimmed, ".torrent") || strings.Contains(trimmed, ".torrent?") {
		return true
	}
	parsed := ParseURLInput(trimmed)
	u, err := url.Parse(parsed.CleanURL)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

func IsPlaylistURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	if u.Query().Has("list") {
		return true
	}
	if strings.Contains(u.Path, "/playlist") || strings.Contains(u.Path, "/sets/") {
		return true
	}
	return false
}
