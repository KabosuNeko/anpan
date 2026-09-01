package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LyricsResult struct {
	ID           int64   `json:"id"`
	TrackName    string  `json:"trackName"`
	ArtistName   string  `json:"artistName"`
	Duration     float64 `json:"duration"`
	PlainLyrics  string  `json:"plainLyrics"`
	SyncedLyrics string  `json:"syncedLyrics"`
}

// FetchLyrics searches LRCLIB for synced or plain lyrics.
func FetchLyrics(ctx context.Context, title, artist string, duration float64) (*LyricsResult, error) {
	cleanTitle := strings.TrimSpace(title)
	cleanArtist := strings.TrimSpace(artist)

	reqCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 4 * time.Second}

	// 1. Try exact match if both artist and title are available
	if cleanArtist != "" && cleanTitle != "" {
		params := url.Values{}
		params.Set("track_name", cleanTitle)
		params.Set("artist_name", cleanArtist)
		if duration > 0 {
			params.Set("duration", fmt.Sprintf("%.0f", duration))
		}
		reqURL := fmt.Sprintf("https://lrclib.net/api/get?%s", params.Encode())
		req, _ := http.NewRequestWithContext(reqCtx, "GET", reqURL, nil)
		if req != nil {
			req.Header.Set("User-Agent", "anpan-downloader/0.5.0 (https://github.com/KabosuNeko/anpan)")
			if resp, err := client.Do(req); err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					var res LyricsResult
					if err := json.NewDecoder(resp.Body).Decode(&res); err == nil && (res.SyncedLyrics != "" || res.PlainLyrics != "") {
						return &res, nil
					}
				}
			}
		}
	}

	// 2. Fallback to search endpoint
	q := cleanTitle
	if cleanArtist != "" && !strings.Contains(strings.ToLower(cleanTitle), strings.ToLower(cleanArtist)) {
		q = cleanArtist + " " + cleanTitle
	}
	searchURL := fmt.Sprintf("https://lrclib.net/api/search?q=%s", url.QueryEscape(q))
	sReq, err := http.NewRequestWithContext(reqCtx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	sReq.Header.Set("User-Agent", "anpan-downloader/0.5.0 (https://github.com/KabosuNeko/anpan)")

	sResp, err := client.Do(sReq)
	if err != nil {
		return nil, err
	}
	defer sResp.Body.Close()

	if sResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lrclib search error: HTTP %d", sResp.StatusCode)
	}

	var results []LyricsResult
	if err := json.NewDecoder(sResp.Body).Decode(&results); err != nil || len(results) == 0 {
		return nil, fmt.Errorf("no lyrics found")
	}

	for _, r := range results {
		if r.SyncedLyrics != "" {
			return &r, nil
		}
	}
	return &results[0], nil
}

// SaveLrcFile writes synced or plain lyrics as a .lrc file alongside the audio file.
func SaveLrcFile(audioPath string, lyrics *LyricsResult) (string, error) {
	if lyrics == nil {
		return "", fmt.Errorf("no lyrics")
	}
	content := lyrics.SyncedLyrics
	if content == "" {
		content = lyrics.PlainLyrics
	}
	if content == "" {
		return "", fmt.Errorf("empty lyrics")
	}

	ext := filepath.Ext(audioPath)
	lrcPath := strings.TrimSuffix(audioPath, ext) + ".lrc"
	if err := os.WriteFile(lrcPath, []byte(content), 0o644); err != nil {
		return "", err
	}
	return lrcPath, nil
}
