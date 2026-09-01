package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
)

var (
	yandereRegex   = regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?yande\.re/post/show/(\d+)`)
	konachanRegex  = regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?konachan\.(?:com|net)/post/show/(\d+)`)
	safebooruRegex = regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?safebooru\.org/(?:index\.php\?page=post&s=view&id=|posts/)(\d+)`)
	gelbooruRegex  = regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?gelbooru\.com/(?:index\.php\?page=post&s=view&id=|posts/)(\d+)`)
)

func IsBooruURL(rawURL string) bool {
	t := strings.TrimSpace(rawURL)
	return yandereRegex.MatchString(t) ||
		konachanRegex.MatchString(t) ||
		safebooruRegex.MatchString(t) ||
		gelbooruRegex.MatchString(t)
}

func ProbeBooruPost(ctx context.Context, rawURL string) (*ArchivePost, error) {
	trimmed := strings.TrimSpace(rawURL)
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 10 * time.Second}

	var apiURL string
	var service string
	var postID string

	if m := yandereRegex.FindStringSubmatch(trimmed); len(m) > 1 {
		service = "yandere"
		postID = m[1]
		apiURL = fmt.Sprintf("https://yande.re/post.json?tags=id:%s", postID)
	} else if m := konachanRegex.FindStringSubmatch(trimmed); len(m) > 1 {
		service = "konachan"
		postID = m[1]
		apiURL = fmt.Sprintf("https://konachan.com/post.json?tags=id:%s", postID)
	} else if m := safebooruRegex.FindStringSubmatch(trimmed); len(m) > 1 {
		service = "safebooru"
		postID = m[1]
		apiURL = fmt.Sprintf("https://safebooru.org/index.php?page=dapi&s=post&q=index&json=1&id=%s", postID)
	} else if m := gelbooruRegex.FindStringSubmatch(trimmed); len(m) > 1 {
		service = "gelbooru"
		postID = m[1]
		apiURL = fmt.Sprintf("https://gelbooru.com/index.php?page=dapi&s=post&q=index&json=1&id=%s", postID)
	} else {
		return nil, fmt.Errorf("unsupported booru url")
	}

	req, err := http.NewRequestWithContext(reqCtx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "anpan-downloader/0.5.0 (Mozilla/5.0)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s api error: HTTP %d", service, resp.StatusCode)
	}

	var fileURL string
	var author string

	if service == "gelbooru" {
		var gData struct {
			Post []struct {
				FileURL string `json:"file_url"`
				Owner   string `json:"owner"`
			} `json:"post"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&gData); err == nil && len(gData.Post) > 0 {
			fileURL = gData.Post[0].FileURL
			author = gData.Post[0].Owner
		}
	} else {
		var posts []map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&posts); err == nil && len(posts) > 0 {
			p := posts[0]
			if u, ok := p["file_url"].(string); ok {
				fileURL = u
			}
			if a, ok := p["author"].(string); ok {
				author = a
			}
		}
	}

	if fileURL == "" {
		return nil, fmt.Errorf("could not find high-res image url for post %s", postID)
	}

	// Handle relative URLs if any
	if strings.HasPrefix(fileURL, "//") {
		fileURL = "https:" + fileURL
	} else if strings.HasPrefix(fileURL, "/") {
		u, _ := url.Parse(trimmed)
		fileURL = fmt.Sprintf("https://%s%s", u.Host, fileURL)
	}

	ext := path.Ext(fileURL)
	if ext == "" {
		ext = ".jpg"
	}
	if idx := strings.Index(ext, "?"); idx != -1 {
		ext = ext[:idx]
	}

	cleanTitle := fmt.Sprintf("%s_%s", service, postID)
	if author != "" {
		cleanTitle = fmt.Sprintf("[%s] %s_%s", sanitizeFilename(author), service, postID)
	}

	fileName := fmt.Sprintf("%s_%s%s", service, postID, ext)

	return &ArchivePost{
		Title:   cleanTitle,
		Service: service,
		User:    author,
		ID:      postID,
		Files: []ArchiveFile{
			{
				Name: fileName,
				URL:  fileURL,
			},
		},
		WebpageURL: rawURL,
	}, nil
}
