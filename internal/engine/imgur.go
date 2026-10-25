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

var imgurRegex = regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?imgur\.com/(?:a/|gallery/|t/[^/]+/)?([a-zA-Z0-9]{5,8})`)

func IsImgurURL(rawURL string) bool {
	return imgurRegex.MatchString(strings.TrimSpace(rawURL))
}

// isImgurAlbumURL distinguishes /a/, /gallery/ and /t/ pages from single-image URLs.
func isImgurAlbumURL(rawURL string) bool {
	lower := strings.ToLower(rawURL)
	return strings.Contains(lower, "/a/") || strings.Contains(lower, "/gallery/") || strings.Contains(lower, "/t/")
}

func newImgurRequest(ctx context.Context, apiURL string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Client-ID 546c25a59c58ad7")
	req.Header.Set("User-Agent", UserAgent)
	return req, nil
}

func probeImgurImage(ctx context.Context, client *http.Client, id, rawURL string) (*ArchivePost, error) {
	req, err := newImgurRequest(ctx, fmt.Sprintf("https://api.imgur.com/3/image/%s", id))
	if err != nil {
		return nil, err
	}
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
		Data    struct {
			Link  string `json:"link"`
			Title string `json:"title"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	if !res.Success || res.Data.Link == "" {
		return nil, fmt.Errorf("no image found for imgur post")
	}

	ext := ".jpg"
	if u, err := url.Parse(res.Data.Link); err == nil {
		if e := filepath.Ext(u.Path); e != "" {
			ext = e
		}
	}
	title := sanitizeFilename(res.Data.Title)
	if title == "" {
		title = fmt.Sprintf("imgur_%s", id)
	}
	name := sanitizeFilename(res.Data.Title)
	if name == "" {
		name = id
	}
	fn := name + ext

	return &ArchivePost{
		Title:      title,
		Service:    "imgur",
		ID:         id,
		Files:      []ArchiveFile{{Name: fn, URL: res.Data.Link}},
		WebpageURL: rawURL,
	}, nil
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
	if !isImgurAlbumURL(trimmed) {
		return probeImgurImage(reqCtx, client, albumID, rawURL)
	}
	apiURL := fmt.Sprintf("https://api.imgur.com/3/album/%s", albumID)

	req, err := newImgurRequest(reqCtx, apiURL)
	if err != nil {
		return nil, err
	}

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
		Data    struct {
			Title  string `json:"title"`
			Images []struct {
				ID    string `json:"id"`
				Link  string `json:"link"`
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
