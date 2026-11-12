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
	"tiktokv.com",
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

// IsTikTokURL reports whether rawURL belongs to a TikTok share domain.
func IsTikTokURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	for _, h := range []string{"tiktok.com", "tiktokv.com"} {
		if host == h || strings.HasSuffix(host, "."+h) {
			return true
		}
	}
	return false
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

func isYouTubeHost(host string) bool {
	host = strings.ToLower(host)
	return host == "youtu.be" || host == "youtube.com" || strings.HasSuffix(host, ".youtube.com")
}

// HasVideoID reports whether a YouTube URL names one video (watch?v=, youtu.be/<id>, /shorts/<id>),
// even when it also carries a list= parameter.
func HasVideoID(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil || !isYouTubeHost(u.Hostname()) {
		return false
	}
	if strings.EqualFold(u.Hostname(), "youtu.be") {
		return strings.Trim(u.Path, "/") != ""
	}
	return u.Query().Get("v") != "" || strings.HasPrefix(u.Path, "/shorts/")
}

// StripYouTubeMix drops the list of an auto-generated YouTube mix (list=RD…) from a video URL.
// YouTube adds these endless mixes when a video is opened, so the user almost always means the one video.
// Curated YouTube Music lists (RDCLAK…) are real playlists and are kept.
func StripYouTubeMix(rawURL string) string {
	if !HasVideoID(rawURL) {
		return rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := u.Query()
	list := q.Get("list")
	if !strings.HasPrefix(list, "RD") || strings.HasPrefix(list, "RDCLAK") {
		return rawURL
	}
	for _, k := range []string{"list", "index", "start_radio", "pp"} {
		q.Del(k)
	}
	u.RawQuery = q.Encode()
	return u.String()
}
