package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var archiveOrgRegex = regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?archive\.org/details/([a-zA-Z0-9_\-\.]+)`)

func IsArchiveOrgURL(rawURL string) bool {
	return archiveOrgRegex.MatchString(strings.TrimSpace(rawURL))
}

func ProbeArchiveOrg(ctx context.Context, rawURL string) (*ArchivePost, error) {
	trimmed := strings.TrimSpace(rawURL)
	m := archiveOrgRegex.FindStringSubmatch(trimmed)
	if len(m) < 2 {
		return nil, fmt.Errorf("invalid archive.org details url")
	}
	itemID := m[1]

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 10 * time.Second}
	apiURL := fmt.Sprintf("https://archive.org/metadata/%s", itemID)

	req, err := http.NewRequestWithContext(reqCtx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "anpan-downloader/1.0.1")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("archive.org metadata request failed: HTTP %d", resp.StatusCode)
	}

	var res struct {
		Metadata struct {
			Title string `json:"title"`
		} `json:"metadata"`
		Files []struct {
			Name   string `json:"name"`
			Format string `json:"format"`
		} `json:"files"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	if len(res.Files) == 0 {
		return nil, fmt.Errorf("no files found in this archive.org item")
	}

	title := sanitizeFilename(res.Metadata.Title)
	if title == "" {
		title = itemID
	}

	archiveFileURL := func(name string) string {
		return fmt.Sprintf("https://archive.org/download/%s/%s", itemID, url.PathEscape(name))
	}

	buildFiles := func(skipInternal bool) []ArchiveFile {
		var out []ArchiveFile
		for _, f := range res.Files {
			if skipInternal && (f.Format == "Metadata" || f.Format == "Item Tile" || strings.HasSuffix(f.Name, "_meta.xml") || strings.HasSuffix(f.Name, "_files.xml")) {
				continue
			}
			cleanName := strings.TrimPrefix(f.Name, "/")
			if cleanName == "" {
				continue
			}
			out = append(out, ArchiveFile{
				Name: sanitizeFilename(cleanName),
				URL:  archiveFileURL(cleanName),
			})
		}
		return out
	}

	files := buildFiles(true)
	if len(files) == 0 {
		files = buildFiles(false)
	}

	return &ArchivePost{
		Title:      title,
		Service:    "archive.org",
		ID:         itemID,
		Files:      files,
		WebpageURL: rawURL,
	}, nil
}
