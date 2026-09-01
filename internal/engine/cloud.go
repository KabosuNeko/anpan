package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	gdriveRegex         = regexp.MustCompile(`(?i)drive\.google\.com/(?:file/d/|open\?id=|uc\?(?:[^&]+&)*id=)([a-zA-Z0-9_-]+)`)
	pixeldrainRegex     = regexp.MustCompile(`(?i)pixeldrain\.com/u/([a-zA-Z0-9_-]+)`)
	pixeldrainListRegex = regexp.MustCompile(`(?i)pixeldrain\.com/l/([a-zA-Z0-9_-]+)`)
	catboxRegex         = regexp.MustCompile(`(?i)(?:files\.)?catbox\.moe/|litterbox\.catbox\.moe/`)
	mediafireRegex      = regexp.MustCompile(`(?i)mediafire\.com/file/([a-zA-Z0-9]+)`)
	mediafireLinkRegex  = regexp.MustCompile(`(?i)href="(https?://download\d+\.mediafire\.com/[^"]+)"`)
)

type CloudDirectFile struct {
	URL      string
	Filename string
	Size     *int64
}

func IsPixeldrainListURL(rawURL string) bool {
	return pixeldrainListRegex.MatchString(strings.TrimSpace(rawURL))
}

func IsCloudHostURL(rawURL string) bool {
	t := strings.TrimSpace(rawURL)
	return gdriveRegex.MatchString(t) || pixeldrainRegex.MatchString(t) || catboxRegex.MatchString(t) || mediafireRegex.MatchString(t)
}

func ProbePixeldrainList(ctx context.Context, rawURL string) (*ArchivePost, error) {
	trimmed := strings.TrimSpace(rawURL)
	m := pixeldrainListRegex.FindStringSubmatch(trimmed)
	if len(m) < 2 {
		return nil, fmt.Errorf("invalid pixeldrain list url")
	}
	id := m[1]

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 10 * time.Second}
	apiURL := fmt.Sprintf("https://pixeldrain.com/api/list/%s", id)

	req, err := http.NewRequestWithContext(reqCtx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pixeldrain list request failed: HTTP %d", resp.StatusCode)
	}

	var listData struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Files []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Size int64  `json:"size"`
		} `json:"files"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&listData); err != nil {
		return nil, err
	}

	if len(listData.Files) == 0 {
		return nil, fmt.Errorf("no files found in this pixeldrain list")
	}

	title := sanitizeFilename(listData.Title)
	if title == "" {
		title = fmt.Sprintf("pixeldrain_list_%s", id)
	}

	var files []ArchiveFile
	for _, f := range listData.Files {
		fn := sanitizeFilename(f.Name)
		if fn == "" {
			fn = fmt.Sprintf("file_%s", f.ID)
		}
		files = append(files, ArchiveFile{
			Name: fn,
			URL:  fmt.Sprintf("https://pixeldrain.com/api/file/%s?download", f.ID),
		})
	}

	return &ArchivePost{
		Title:      title,
		Service:    "pixeldrain",
		ID:         id,
		Files:      files,
		WebpageURL: rawURL,
	}, nil
}

func ProbeCloudHost(ctx context.Context, rawURL string) (*CloudDirectFile, error) {
	trimmed := strings.TrimSpace(rawURL)
	reqCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 8 * time.Second}

	// 1. Pixeldrain
	if m := pixeldrainRegex.FindStringSubmatch(trimmed); len(m) > 1 {
		id := m[1]
		infoURL := fmt.Sprintf("https://pixeldrain.com/api/file/%s/info", id)
		req, _ := http.NewRequestWithContext(reqCtx, "GET", infoURL, nil)
		if req != nil {
			if resp, err := client.Do(req); err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					var info struct {
						Name string `json:"name"`
						Size int64  `json:"size"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&info); err == nil {
						filename := sanitizeFilename(info.Name)
						if filename == "" {
							filename = fmt.Sprintf("pixeldrain_%s", id)
						}
						size := info.Size
						return &CloudDirectFile{
							URL:      fmt.Sprintf("https://pixeldrain.com/api/file/%s?download", id),
							Filename: filename,
							Size:     &size,
						}, nil
					}
				}
			}
		}
		return &CloudDirectFile{
			URL:      fmt.Sprintf("https://pixeldrain.com/api/file/%s?download", id),
			Filename: fmt.Sprintf("pixeldrain_%s", id),
		}, nil
	}

	// 2. Google Drive
	if m := gdriveRegex.FindStringSubmatch(trimmed); len(m) > 1 {
		id := m[1]
		directURL := fmt.Sprintf("https://drive.google.com/uc?export=download&id=%s&confirm=t", id)

		req, _ := http.NewRequestWithContext(reqCtx, "HEAD", directURL, nil)
		if req != nil {
			if resp, err := client.Do(req); err == nil {
				defer resp.Body.Close()
				filename := ""
				disp := resp.Header.Get("Content-Disposition")
				if disp != "" {
					if idx := strings.Index(disp, "filename="); idx != -1 {
						fn := strings.Trim(disp[idx+9:], `"'; `)
						if fn != "" {
							filename = sanitizeFilename(fn)
						}
					}
				}
				if filename == "" {
					filename = fmt.Sprintf("gdrive_%s", id)
				}
				var size *int64
				if resp.ContentLength > 0 {
					s := resp.ContentLength
					size = &s
				}
				return &CloudDirectFile{
					URL:      directURL,
					Filename: filename,
					Size:     size,
				}, nil
			}
		}
		return &CloudDirectFile{
			URL:      directURL,
			Filename: fmt.Sprintf("gdrive_%s", id),
		}, nil
	}

	// 3. Catbox / Litterbox
	if catboxRegex.MatchString(trimmed) {
		u, err := url.Parse(trimmed)
		filename := "catbox_file"
		if err == nil && u.Path != "" {
			parts := strings.Split(u.Path, "/")
			if len(parts) > 0 && parts[len(parts)-1] != "" {
				filename = sanitizeFilename(parts[len(parts)-1])
			}
		}
		return &CloudDirectFile{
			URL:      trimmed,
			Filename: filename,
		}, nil
	}

	// 4. MediaFire
	if mediafireRegex.MatchString(trimmed) {
		req, _ := http.NewRequestWithContext(reqCtx, "GET", trimmed, nil)
		if req != nil {
			req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64)")
			if resp, err := client.Do(req); err == nil {
				defer resp.Body.Close()
				data, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
				html := string(data)
				if m := mediafireLinkRegex.FindStringSubmatch(html); len(m) > 1 {
					directLink := m[1]
					filename := "mediafire_file"
					if u, err := url.Parse(directLink); err == nil {
						parts := strings.Split(u.Path, "/")
						if len(parts) > 0 && parts[len(parts)-1] != "" {
							filename = sanitizeFilename(parts[len(parts)-1])
						}
					}
					return &CloudDirectFile{
						URL:      directLink,
						Filename: filename,
					}, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("unsupported cloud host")
}
