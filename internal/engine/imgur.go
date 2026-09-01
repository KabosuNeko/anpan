package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var imgurRegex = regexp.MustCompile(`(?i)imgur\.com/(?:a/|gallery/|t/[^/]+/)?([a-zA-Z0-9]{5,8})`)

func IsImgurURL(rawURL string) bool {
	t := strings.TrimSpace(rawURL)
	return imgurRegex.MatchString(t) && !strings.Contains(t, "i.imgur.com")
}

func ProbeImgurAlbum(ctx context.Context, rawURL string) (*ArchivePost, error) {
	trimmed := strings.TrimSpace(rawURL)
	m := imgurRegex.FindStringSubmatch(trimmed)
	if len(m) < 2 {
		return nil, fmt.Errorf("invalid imgur url")
	}
	albumID := m[1]

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 10 * time.Second}
	apiURL := fmt.Sprintf("https://api.imgur.com/3/album/%s", albumID)

	req, err := http.NewRequestWithContext(reqCtx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Client-ID 546c25a59c58ad7")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("imgur api error: HTTP %d", resp.StatusCode)
	}

	var res struct {
		Success bool `json:"success"`
		Status  int  `json:"status"`
		Data    struct {
			ID     string `json:"id"`
			Title  string `json:"title"`
			Images []struct {
				ID    string `json:"id"`
				Link  string `json:"link"`
				Type  string `json:"type"`
				Size  int64  `json:"size"`
				Title string `json:"title"`
			} `json:"images"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	if !res.Success || len(res.Data.Images) == 0 {
		return nil, fmt.Errorf("no images found in imgur album")
	}

	title := sanitizeFilename(res.Data.Title)
	if title == "" {
		title = fmt.Sprintf("imgur_%s", albumID)
	}

	var files []ArchiveFile
	for i, img := range res.Data.Images {
		link := img.Link
		ext := ".jpg"
		if u, err := url.Parse(link); err == nil {
			if e := filepath.Ext(u.Path); e != "" {
				ext = e
			}
		} else if img.Type == "image/png" {
			ext = ".png"
		} else if img.Type == "image/gif" {
			ext = ".gif"
		} else if img.Type == "video/mp4" {
			ext = ".mp4"
		}

		fn := fmt.Sprintf("%02d_%s%s", i+1, img.ID, ext)
		if img.Title != "" {
			if clean := sanitizeFilename(img.Title); clean != "" {
				fn = fmt.Sprintf("%02d_%s%s", i+1, clean, ext)
			}
		}

		files = append(files, ArchiveFile{
			Name: fn,
			URL:  link,
		})
	}

	return &ArchivePost{
		Title:      title,
		Service:    "imgur",
		ID:         albumID,
		Files:      files,
		WebpageURL: rawURL,
	}, nil
}
