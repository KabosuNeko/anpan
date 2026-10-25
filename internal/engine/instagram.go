package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var instagramPostRegex = regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?instagram\.com/p/([a-zA-Z0-9_-]+)`)

// IsInstagramPostURL reports whether rawURL is an Instagram /p/ post (single media or carousel).
func IsInstagramPostURL(rawURL string) bool {
	return instagramPostRegex.MatchString(strings.TrimSpace(rawURL))
}

type instagramMedia struct {
	Display string `json:"display_url"`
	Video   string `json:"video_url"`
	Sidecar *struct {
		Edges []struct {
			Node instagramMedia `json:"node"`
		} `json:"edges"`
	} `json:"edge_sidecar_to_children"`
	Caption struct {
		Edges []struct {
			Node struct {
				Text string `json:"text"`
			} `json:"node"`
		} `json:"edges"`
	} `json:"edge_media_to_caption"`
	Owner struct {
		Username string `json:"username"`
	} `json:"owner"`
}

var instagramInitRegex = regexp.MustCompile(`(?s)"init",\[\],\[(.*?)\]\],`)

// ProbeInstagramPost reads the public embed JSON, which lists sidecar children for carousel posts.
func ProbeInstagramPost(ctx context.Context, rawURL string) (*ArchivePost, error) {
	m := instagramPostRegex.FindStringSubmatch(strings.TrimSpace(rawURL))
	if len(m) < 2 {
		return nil, fmt.Errorf("invalid instagram post url")
	}
	id := m[1]

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	endpoint := fmt.Sprintf("https://www.instagram.com/p/%s/embed/captioned/", id)
	req, err := http.NewRequestWithContext(reqCtx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", browserUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "en-GB,en;q=0.9")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Dnt", "1")
	req.Header.Set("Priority", "u=0, i")
	req.Header.Set("Sec-Ch-Ua", `Chromium";v="124", "Google Chrome";v="124", "Not-A.Brand";v="99"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", "macOS")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("instagram embed request failed: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	return parseInstagramEmbedHTML(body, id, rawURL)
}

func parseInstagramEmbedHTML(html []byte, id, rawURL string) (*ArchivePost, error) {
	m := instagramInitRegex.FindSubmatch(html)
	if len(m) < 2 {
		return nil, fmt.Errorf("instagram post data not found")
	}

	var init []struct {
		ContextJSON string `json:"contextJSON"`
	}
	if err := json.Unmarshal(append(append([]byte("["), m[1]...), ']'), &init); err != nil {
		return nil, fmt.Errorf("parse instagram embed: %w", err)
	}
	if len(init) == 0 || init[0].ContextJSON == "" {
		return nil, fmt.Errorf("instagram post data not found")
	}

	var context struct {
		GQLData struct {
			ShortcodeMedia    *instagramMedia `json:"shortcode_media"`
			XdtShortcodeMedia *instagramMedia `json:"xdt_shortcode_media"`
		} `json:"gql_data"`
		ShortcodeMedia *instagramMedia `json:"shortcode_media"`
	}
	if err := json.Unmarshal([]byte(init[0].ContextJSON), &context); err != nil {
		return nil, fmt.Errorf("parse instagram post: %w", err)
	}

	media := context.GQLData.ShortcodeMedia
	if media == nil {
		media = context.GQLData.XdtShortcodeMedia
	}
	if media == nil {
		media = context.ShortcodeMedia
	}
	if media == nil {
		return nil, fmt.Errorf("instagram post data not found")
	}

	files := instagramMediaFiles(media, id)
	if len(files) == 0 {
		return nil, fmt.Errorf("no media found in this instagram post")
	}

	title := "instagram_" + id
	if len(media.Caption.Edges) > 0 {
		if caption := sanitizeFilename(media.Caption.Edges[0].Node.Text); caption != "" {
			title = caption
		}
	}

	return &ArchivePost{
		Title:      title,
		Service:    "instagram",
		User:       media.Owner.Username,
		ID:         id,
		Files:      files,
		WebpageURL: rawURL,
	}, nil
}

func instagramMediaFiles(media *instagramMedia, id string) []ArchiveFile {
	if media.Sidecar != nil && len(media.Sidecar.Edges) > 0 {
		var files []ArchiveFile
		for i, edge := range media.Sidecar.Edges {
			node := edge.Node
			switch {
			case node.Video != "":
				files = append(files, ArchiveFile{
					Name: fmt.Sprintf("instagram_%s_%02d.mp4", id, i+1),
					URL:  node.Video,
				})
			case node.Display != "":
				files = append(files, ArchiveFile{
					Name: fmt.Sprintf("instagram_%s_%02d.jpg", id, i+1),
					URL:  node.Display,
				})
			}
		}
		return files
	}

	if media.Video != "" {
		return []ArchiveFile{{Name: fmt.Sprintf("instagram_%s.mp4", id), URL: media.Video}}
	}
	if media.Display != "" {
		return []ArchiveFile{{Name: fmt.Sprintf("instagram_%s.jpg", id), URL: media.Display}}
	}
	return nil
}
