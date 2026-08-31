package core

import (
	"net/url"
	"regexp"
	"strings"
)

type SiteInfo struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type knownSite struct {
	hosts []string
	site  SiteInfo
}

var knownSites = []knownSite{
	{hosts: []string{"youtube.com", "youtu.be", "music.youtube.com"}, site: SiteInfo{Key: "youtube", Label: "YouTube"}},
	{hosts: []string{"x.com", "twitter.com"}, site: SiteInfo{Key: "x", Label: "X / Twitter"}},
	{hosts: []string{"instagram.com"}, site: SiteInfo{Key: "instagram", Label: "Instagram"}},
	{hosts: []string{"threads.net", "threads.com"}, site: SiteInfo{Key: "threads", Label: "Threads"}},
	{hosts: []string{"tiktok.com"}, site: SiteInfo{Key: "tiktok", Label: "TikTok"}},
	{hosts: []string{"vimeo.com"}, site: SiteInfo{Key: "vimeo", Label: "Vimeo"}},
	{hosts: []string{"twitch.tv"}, site: SiteInfo{Key: "twitch", Label: "Twitch"}},
	{hosts: []string{"reddit.com"}, site: SiteInfo{Key: "reddit", Label: "Reddit"}},
	{hosts: []string{"facebook.com", "fb.watch"}, site: SiteInfo{Key: "facebook", Label: "Facebook"}},
	{hosts: []string{"soundcloud.com"}, site: SiteInfo{Key: "soundcloud", Label: "SoundCloud"}},
	{hosts: []string{"bandcamp.com"}, site: SiteInfo{Key: "bandcamp", Label: "Bandcamp"}},
	{hosts: []string{"kemono.cr", "kemono.su", "kemono.party"}, site: SiteInfo{Key: "kemono", Label: "Kemono"}},
	{hosts: []string{"coomer.su", "coomer.party", "coomer.st"}, site: SiteInfo{Key: "coomer", Label: "Coomer"}},
	{hosts: []string{"pawchive.st", "pawchive.pw"}, site: SiteInfo{Key: "pawchive", Label: "Pawchive"}},
}

func IdentifySite(rawURL string) SiteInfo {
	u, err := url.Parse(rawURL)
	if err != nil || u.Hostname() == "" {
		return SiteInfo{Key: "unknown", Label: "Unknown site"}
	}
	hostname := strings.ToLower(u.Hostname())

	for _, ks := range knownSites {
		for _, h := range ks.hosts {
			if hostname == h || strings.HasSuffix(hostname, "."+h) {
				return ks.site
			}
		}
	}

	return SiteInfo{Key: "generic", Label: hostname}
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
