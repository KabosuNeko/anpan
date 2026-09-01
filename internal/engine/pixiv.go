package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"
)

var pixivRegex = regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?pixiv\.(?:net|me)/(?:(?:en/)?artworks/|i/)(\d+)`)

func IsPixivURL(rawURL string) bool {
	return pixivRegex.MatchString(strings.TrimSpace(rawURL))
}

func ParsePixivID(rawURL string) string {
	m := pixivRegex.FindStringSubmatch(strings.TrimSpace(rawURL))
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

func ProbePixivPost(ctx context.Context, rawURL string) (*ArchivePost, error) {
	id := ParsePixivID(rawURL)
	if id == "" {
		return nil, fmt.Errorf("invalid pixiv url")
	}

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 10 * time.Second}

	illustURL := fmt.Sprintf("https://www.pixiv.net/ajax/illust/%s", id)
	req, err := http.NewRequestWithContext(reqCtx, "GET", illustURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://www.pixiv.net/")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pixiv illust request failed: HTTP %d", resp.StatusCode)
	}

	var illustData struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
		Body    struct {
			Title     string `json:"title"`
			UserName  string `json:"userName"`
			PageCount int    `json:"pageCount"`
			Urls      struct {
				Original string `json:"original"`
			} `json:"urls"`
		} `json:"body"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&illustData); err != nil {
		return nil, err
	}
	if illustData.Error {
		return nil, fmt.Errorf("pixiv api error: %s", illustData.Message)
	}

	title := sanitizeFilename(illustData.Body.Title)
	if title == "" {
		title = fmt.Sprintf("pixiv_%s", id)
	}
	user := sanitizeFilename(illustData.Body.UserName)
	if user == "" {
		user = "pixiv_artist"
	}

	var files []ArchiveFile

	if illustData.Body.PageCount > 1 {
		pagesURL := fmt.Sprintf("https://www.pixiv.net/ajax/illust/%s/pages", id)
		pReq, err := http.NewRequestWithContext(reqCtx, "GET", pagesURL, nil)
		if err == nil {
			pReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
			pReq.Header.Set("Referer", "https://www.pixiv.net/")
			if pResp, err := client.Do(pReq); err == nil {
				defer pResp.Body.Close()
				var pagesData struct {
					Error bool `json:"error"`
					Body  []struct {
						Urls struct {
							Original string `json:"original"`
						} `json:"urls"`
					} `json:"body"`
				}
				if err := json.NewDecoder(pResp.Body).Decode(&pagesData); err == nil && !pagesData.Error {
					for i, page := range pagesData.Body {
						origURL := page.Urls.Original
						if origURL != "" {
							ext := path.Ext(origURL)
							if ext == "" {
								ext = ".jpg"
							}
							filename := fmt.Sprintf("%s_p%d%s", title, i, ext)
							files = append(files, ArchiveFile{
								Name:    filename,
								URL:     origURL,
								Headers: []string{"Referer: https://www.pixiv.net/", "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"},
							})
						}
					}
				}
			}
		}
	}

	if len(files) == 0 && illustData.Body.Urls.Original != "" {
		origURL := illustData.Body.Urls.Original
		ext := path.Ext(origURL)
		if ext == "" {
			ext = ".jpg"
		}
		filename := fmt.Sprintf("%s_p0%s", title, ext)
		files = append(files, ArchiveFile{
			Name:    filename,
			URL:     origURL,
			Headers: []string{"Referer: https://www.pixiv.net/", "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"},
		})
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no original images found for pixiv illust %s", id)
	}

	return &ArchivePost{
		Title:      fmt.Sprintf("[%s] %s", user, title),
		Service:    "pixiv",
		User:       user,
		ID:         id,
		Files:      files,
		WebpageURL: rawURL,
	}, nil
}
