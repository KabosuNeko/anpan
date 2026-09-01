package engine

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
)

type ArchiveFile struct {
	Name    string   `json:"name"`
	URL     string   `json:"url"`
	Mirrors []string `json:"mirrors,omitempty"`
	Headers []string `json:"headers,omitempty"`
}

type ArchivePost struct {
	Title      string        `json:"title"`
	Service    string        `json:"service"`
	User       string        `json:"user"`
	ID         string        `json:"id"`
	Files      []ArchiveFile `json:"files"`
	WebpageURL string        `json:"webpage_url"`
}

var archivePostRegex = regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?(kemono\.(?:cr|su|party)|coomer\.(?:su|party|st)|pawchive\.(?:st|pw))/(?:api/v1/)?([^/?#]+)/user/([^/?#]+)/post/([^/?#]+)`)

func IsArchivePostURL(rawURL string) bool {
	return archivePostRegex.MatchString(strings.TrimSpace(rawURL))
}

type ParsedArchiveURL struct {
	Domain  string
	Service string
	User    string
	ID      string
}

func ParseArchiveURL(rawURL string) *ParsedArchiveURL {
	match := archivePostRegex.FindStringSubmatch(strings.TrimSpace(rawURL))
	if match == nil {
		return nil
	}
	return &ParsedArchiveURL{
		Domain:  strings.ToLower(match[1]),
		Service: match[2],
		User:    match[3],
		ID:      match[4],
	}
}

var invalidCharRegex = regexp.MustCompile(`[/\\?%*:|"<>]+`)

func sanitizeFilename(name string) string {
	return invalidCharRegex.ReplaceAllString(name, "_")
}

func ProbeArchivePost(ctx context.Context, rawURL string) (*ArchivePost, error) {
	parsed := ParseArchiveURL(rawURL)
	if parsed == nil {
		return nil, fmt.Errorf("not a valid archive post url")
	}

	isPawchive := strings.HasPrefix(parsed.Domain, "pawchive")
	isCoomer := strings.HasPrefix(parsed.Domain, "coomer")

	apiHost := "https://kemono.cr"
	if isPawchive {
		apiHost = "https://pawchive.pw"
	} else if isCoomer {
		apiHost = "https://coomer.st"
	}

	endpoint := fmt.Sprintf("%s/api/v1/%s/user/%s/post/%s", apiHost, parsed.Service, parsed.User, parsed.ID)

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	if isPawchive {
		req.Header.Set("Accept", "application/json")
	} else {
		req.Header.Set("Accept", "text/css")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("archive api returned %s", resp.Status)
	}

	var reader io.Reader = resp.Body
	if strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") {
		gzReader, err := gzip.NewReader(resp.Body)
		if err == nil {
			defer gzReader.Close()
			reader = gzReader
		}
	} else {
		buf := bufio.NewReader(resp.Body)
		peek, _ := buf.Peek(2)
		if len(peek) == 2 && peek[0] == 0x1f && peek[1] == 0x8b {
			gzReader, err := gzip.NewReader(buf)
			if err == nil {
				defer gzReader.Close()
				reader = gzReader
			} else {
				reader = buf
			}
		} else {
			reader = buf
		}
	}

	var rawData map[string]any
	if err := json.NewDecoder(reader).Decode(&rawData); err != nil {
		return nil, err
	}

	postObj := rawData
	if p, ok := rawData["post"].(map[string]any); ok {
		postObj = p
	}

	var files []ArchiveFile
	seenPaths := make(map[string]bool)

	addFile := func(fileData any) {
		m, ok := fileData.(map[string]any)
		if !ok {
			return
		}
		filePath, _ := m["path"].(string)
		if filePath == "" || seenPaths[filePath] {
			return
		}
		seenPaths[filePath] = true

		fileName, _ := m["name"].(string)
		if fileName == "" {
			fileName = path.Base(filePath)
			if fileName == "" || fileName == "." || fileName == "/" {
				fileName = "file"
			}
		}
		cleanName := sanitizeFilename(fileName)
		encodedName := url.QueryEscape(cleanName)

		var mirrors []string
		if isCoomer {
			mirrors = append(mirrors, fmt.Sprintf("https://coomer.st/data%s?f=%s", filePath, encodedName))
		} else {
			mirrors = append(mirrors, fmt.Sprintf("https://file.pawchive.pw/data%s?f=%s", filePath, encodedName))
			mirrors = append(mirrors, fmt.Sprintf("https://kemono.cr/data%s?f=%s", filePath, encodedName))
		}

		files = append(files, ArchiveFile{
			Name:    cleanName,
			URL:     mirrors[0],
			Mirrors: mirrors,
		})
	}

	if postFile, ok := postObj["file"]; ok {
		addFile(postFile)
	}
	if attachments, ok := postObj["attachments"].([]any); ok {
		for _, att := range attachments {
			addFile(att)
		}
	}

	rawTitle, _ := postObj["title"].(string)
	if strings.TrimSpace(rawTitle) == "" {
		rawTitle = fmt.Sprintf("%s post by %s", parsed.Service, parsed.User)
	}
	cleanTitle := sanitizeFilename(strings.TrimSpace(rawTitle))
	if cleanTitle == "" {
		cleanTitle = "Archive Post"
	}

	return &ArchivePost{
		Title:      cleanTitle,
		Service:    parsed.Service,
		User:       parsed.User,
		ID:         parsed.ID,
		Files:      files,
		WebpageURL: rawURL,
	}, nil
}
