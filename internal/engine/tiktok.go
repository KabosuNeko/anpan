package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const tiktokUniversalMarker = `<script id="__UNIVERSAL_DATA_FOR_REHYDRATION__" type="application/json">`

type tiktokPageData struct {
	DefaultScope struct {
		VideoDetail struct {
			StatusMsg string `json:"statusMsg"`
			ItemInfo  struct {
				ItemStruct struct {
					ID     string `json:"id"`
					Desc   string `json:"desc"`
					Author struct {
						UniqueID string `json:"uniqueId"`
					} `json:"author"`
					ImagePost struct {
						Title  string `json:"title"`
						Images []struct {
							ImageURL struct {
								URLList []string `json:"urlList"`
							} `json:"imageURL"`
						} `json:"images"`
					} `json:"imagePost"`
				} `json:"itemStruct"`
			} `json:"itemInfo"`
		} `json:"webapp.video-detail"`
	} `json:"__DEFAULT_SCOPE__"`
}

// ProbeTikTokPhotos returns photo-slide posts. yt-dlp solves TikTok's JS challenge,
// so its saved page dump is the only reliable place to read imagePost data.
func ProbeTikTokPhotos(ctx context.Context, ytdlpBin, rawURL string) (*ArchivePost, error) {
	dir, err := os.MkdirTemp("", "anpan-tiktok-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	cmd := exec.CommandContext(ctx, ytdlpBin,
		"--write-pages", "--skip-download", "--ignore-no-formats-error", "--no-warnings", "--no-playlist", rawURL)
	cmd.Dir = dir
	cmd.Stdout = io.Discard
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	runErr := cmd.Run()

	dumps, _ := filepath.Glob(filepath.Join(dir, "*.dump"))
	for _, dump := range dumps {
		data, readErr := os.ReadFile(dump)
		if readErr != nil {
			continue
		}
		post, parseErr := parseTikTokPhotoPage(data, rawURL)
		if parseErr == nil && post != nil {
			return post, nil
		}
	}

	if runErr != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("tiktok: %s", msg)
		}
		return nil, runErr
	}
	return nil, nil
}

func parseTikTokPhotoPage(content []byte, rawURL string) (*ArchivePost, error) {
	start := bytes.Index(content, []byte(tiktokUniversalMarker))
	if start < 0 {
		return nil, fmt.Errorf("tiktok page data not found")
	}
	start += len(tiktokUniversalMarker)
	end := bytes.Index(content[start:], []byte("</script>"))
	if end < 0 {
		return nil, fmt.Errorf("tiktok page data is incomplete")
	}

	var data tiktokPageData
	if err := json.Unmarshal(content[start:start+end], &data); err != nil {
		return nil, fmt.Errorf("parse tiktok page: %w", err)
	}

	detail := data.DefaultScope.VideoDetail
	if detail.StatusMsg != "" {
		return nil, fmt.Errorf("tiktok: %s", detail.StatusMsg)
	}
	item := detail.ItemInfo.ItemStruct
	if len(item.ImagePost.Images) == 0 {
		return nil, nil
	}

	id := item.ID
	if id == "" {
		id = tiktokIDFromURL(rawURL)
	}
	author := sanitizeFilename(item.Author.UniqueID)
	if author == "" {
		author = "tiktok"
	}

	title := sanitizeFilename(item.ImagePost.Title)
	if title == "" {
		title = sanitizeFilename(item.Desc)
	}
	if title == "" {
		title = "tiktok_" + id
	}

	var files []ArchiveFile
	for i, image := range item.ImagePost.Images {
		url := pickTikTokImageURL(image.ImageURL.URLList)
		if url == "" {
			continue
		}
		files = append(files, ArchiveFile{
			Name: fmt.Sprintf("tiktok_%s_%s_photo_%02d.jpg", author, id, i+1),
			URL:  url,
		})
	}
	if len(files) == 0 {
		return nil, nil
	}

	return &ArchivePost{
		Title:      title,
		Service:    "tiktok",
		User:       author,
		ID:         id,
		Files:      files,
		WebpageURL: rawURL,
	}, nil
}

func pickTikTokImageURL(urls []string) string {
	for _, u := range urls {
		if strings.Contains(u, ".jpeg?") {
			return u
		}
	}
	for _, u := range urls {
		if u != "" {
			return u
		}
	}
	return ""
}

func tiktokIDFromURL(rawURL string) string {
	for _, marker := range []string{"/video/", "/photo/"} {
		if i := strings.LastIndex(rawURL, marker); i >= 0 {
			id := rawURL[i+len(marker):]
			if j := strings.IndexAny(id, "/?#"); j >= 0 {
				id = id[:j]
			}
			if id != "" {
				return sanitizeFilename(id)
			}
		}
	}
	return "post"
}
