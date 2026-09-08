package engine

import (
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// DefaultPublicTrackers contains reliable Tier-1 public BitTorrent trackers
var DefaultPublicTrackers = []string{
	"udp://tracker.opentrackr.org:1337/announce",
	"udp://open.demonii.com:1337/announce",
	"udp://open.stealth.si:80/announce",
	"udp://tracker.torrent.eu.org:451/announce",
	"udp://explodie.org:6969/announce",
	"udp://tracker.openbittorrent.com:6969/announce",
	"udp://tracker.dler.org:6969/announce",
	"http://tracker.openbittorrent.com:80/announce",
}

var (
	hexRegex    = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)
	base32Regex = regexp.MustCompile(`^[2-7a-zA-Z]{32}$`)
)

// IsInfoHash returns true if the input string is a valid 40-char hex or 32-char Base32 BitTorrent InfoHash.
func IsInfoHash(s string) bool {
	trimmed := strings.TrimSpace(s)
	return hexRegex.MatchString(trimmed) || base32Regex.MatchString(trimmed)
}

// NormalizeInfoHash converts a 40-hex or 32-base32 InfoHash to a 40-char lowercase hex string.
func NormalizeInfoHash(s string) (string, error) {
	trimmed := strings.TrimSpace(s)
	if hexRegex.MatchString(trimmed) {
		return strings.ToLower(trimmed), nil
	}
	if base32Regex.MatchString(trimmed) {
		b32 := strings.ToUpper(trimmed)
		bytes, err := base32.StdEncoding.DecodeString(b32)
		if err != nil {
			return "", fmt.Errorf("invalid base32 infohash: %w", err)
		}
		return hex.EncodeToString(bytes), nil
	}
	return "", fmt.Errorf("not a valid infohash: %q", s)
}

// BuildMagnet constructs a magnet URI with default public trackers.
func BuildMagnet(infoHash, name string, customTrackers []string) string {
	ih, err := NormalizeInfoHash(infoHash)
	if err != nil {
		ih = strings.TrimSpace(infoHash)
	}

	var sb strings.Builder
	sb.WriteString("magnet:?xt=urn:btih:")
	sb.WriteString(ih)

	if name != "" {
		sb.WriteString("&dn=")
		sb.WriteString(url.QueryEscape(name))
	}

	trackers := DefaultPublicTrackers
	if len(customTrackers) > 0 {
		trackers = customTrackers
	}

	for _, tr := range trackers {
		sb.WriteString("&tr=")
		sb.WriteString(url.QueryEscape(tr))
	}

	return sb.String()
}

// EnrichMagnetWithTrackers ensures a magnet URI has essential Tier-1 trackers attached for rapid DHT discovery.
func EnrichMagnetWithTrackers(magnetURI string) string {
	if !strings.HasPrefix(strings.ToLower(magnetURI), "magnet:?") {
		return magnetURI
	}

	u, err := url.Parse(magnetURI)
	if err != nil {
		return magnetURI
	}

	q := u.Query()
	existingTrackers := make(map[string]bool)
	for _, tr := range q["tr"] {
		existingTrackers[strings.ToLower(tr)] = true
	}

	added := false
	for _, tr := range DefaultPublicTrackers {
		if !existingTrackers[strings.ToLower(tr)] {
			q.Add("tr", tr)
			added = true
		}
	}

	if !added {
		return magnetURI
	}

	u.RawQuery = q.Encode()
	return u.String()
}

// ParseInfoHashFromMagnet extracts the 40-char hex InfoHash from a magnet URI.
func ParseInfoHashFromMagnet(magnetURI string) string {
	lower := strings.ToLower(magnetURI)
	idx := strings.Index(lower, "urn:btih:")
	if idx == -1 {
		return ""
	}
	sub := magnetURI[idx+9:]
	end := strings.IndexAny(sub, "&/?# ")
	if end != -1 {
		sub = sub[:end]
	}
	norm, err := NormalizeInfoHash(sub)
	if err == nil {
		return norm
	}
	return sub
}
