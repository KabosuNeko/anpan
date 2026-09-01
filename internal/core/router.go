package core

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/KabosuNeko/anpan/internal/engine"
)

type TargetType string

const (
	TargetTorrent TargetType = "torrent"
	TargetDirect  TargetType = "direct"
	TargetArchive TargetType = "archive"
	TargetVideo   TargetType = "video"
)

type TargetInspection struct {
	Type TargetType

	// For torrent
	Target string
	Name   string

	// For direct
	URL      string
	Filename string
	Size     *int64

	// For archive
	ArchivePost *engine.ArchivePost

	// For video
	CleanURL  string
	TimeRange string
	TimeLabel string
}

var directExtensions = map[string]bool{
	"iso": true, "img": true, "zip": true, "tar": true, "gz": true,
	"bz2": true, "xz": true, "7z": true, "rar": true, "tgz": true,
	"bin": true, "pkg": true, "deb": true, "rpm": true, "appimage": true,
	"exe": true, "dmg": true, "pdf": true, "epub": true, "apk": true,
	"jar": true,
}

func ParseMagnetName(magnetURI string) string {
	u, err := url.Parse(magnetURI)
	if err != nil {
		return "BitTorrent Transfer"
	}
	dn := u.Query().Get("dn")
	if dn != "" {
		return strings.ReplaceAll(dn, "+", " ")
	}
	xt := u.Query().Get("xt")
	if xt != "" {
		parts := strings.Split(xt, ":")
		hash := parts[len(parts)-1]
		if len(hash) > 10 {
			hash = hash[:10]
		}
		return "Torrent (" + hash + ")"
	}
	return "BitTorrent Transfer"
}

func extractFilenameFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "download"
	}
	base := path.Base(u.Path)
	if base == "" || base == "." || base == "/" {
		return "download"
	}
	return base
}

func InspectTarget(ctx context.Context, rawInput string) (*TargetInspection, error) {
	trimmed := strings.TrimSpace(rawInput)

	if strings.HasPrefix(trimmed, "magnet:?") {
		return &TargetInspection{
			Type:   TargetTorrent,
			Target: trimmed,
			Name:   ParseMagnetName(trimmed),
		}, nil
	}

	if strings.HasSuffix(trimmed, ".torrent") || strings.Contains(trimmed, ".torrent?") {
		parts := strings.Split(trimmed, "?")
		name := strings.TrimSuffix(path.Base(parts[0]), ".torrent")
		return &TargetInspection{
			Type:   TargetTorrent,
			Target: trimmed,
			Name:   name,
		}, nil
	}

	parsedInput := ParseURLInput(trimmed)
	cleanURL := parsedInput.CleanURL

	if engine.IsArchivePostURL(cleanURL) {
		archive, err := engine.ProbeArchivePost(ctx, cleanURL)
		if err != nil {
			return nil, fmt.Errorf("archive post could not be loaded: %w", err)
		}
		if archive != nil && len(archive.Files) > 0 {
			return &TargetInspection{
				Type:        TargetArchive,
				ArchivePost: archive,
			}, nil
		}
		return nil, fmt.Errorf("no downloadable files or attachments found in this post")
	}

	site := IdentifySite(cleanURL)
	if parsedInput.TimeRange != "" || site.Key != "generic" {
		return &TargetInspection{
			Type:      TargetVideo,
			CleanURL:  cleanURL,
			TimeRange: parsedInput.TimeRange,
			TimeLabel: parsedInput.TimeLabel,
		}, nil
	}

	u, err := url.Parse(cleanURL)
	if err != nil {
		return &TargetInspection{
			Type:     TargetVideo,
			CleanURL: cleanURL,
		}, nil
	}

	pathname := strings.ToLower(u.Path)
	ext := strings.TrimPrefix(path.Ext(pathname), ".")
	if directExtensions[ext] {
		return &TargetInspection{
			Type:     TargetDirect,
			URL:      cleanURL,
			Filename: path.Base(pathname),
		}, nil
	}

	headCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(headCtx, "HEAD", cleanURL, nil)
	if err == nil {
		client := &http.Client{Timeout: 1500 * time.Millisecond}
		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				disposition := resp.Header.Get("Content-Disposition")
				contentType := strings.ToLower(resp.Header.Get("Content-Type"))
				lengthHeader := resp.Header.Get("Content-Length")
				var size *int64
				if lengthHeader != "" {
					if s, err := strconv.ParseInt(lengthHeader, 10, 64); err == nil {
						size = &s
					}
				}

				dispositionFilename := ""
				if disposition != "" {
					if _, params, err := mime.ParseMediaType(disposition); err == nil {
						dispositionFilename = params["filename"]
					}
				}

				isAttachment := strings.Contains(disposition, "attachment")
				isBinary := strings.Contains(contentType, "application/octet-stream") ||
					strings.Contains(contentType, "application/x-") ||
					strings.Contains(contentType, "application/zip") ||
					(!strings.Contains(contentType, "text/html") && !strings.Contains(contentType, "application/xhtml+xml"))

				if (isAttachment || isBinary) && (dispositionFilename != "" || (size != nil && *size > 1_000_000)) {
					filename := dispositionFilename
					if filename == "" {
						filename = extractFilenameFromURL(cleanURL)
					}
					return &TargetInspection{
						Type:     TargetDirect,
						URL:      cleanURL,
						Filename: filename,
						Size:     size,
					}, nil
				}
			}
		}
	}

	return &TargetInspection{
		Type:      TargetVideo,
		CleanURL:  cleanURL,
		TimeRange: parsedInput.TimeRange,
		TimeLabel: parsedInput.TimeLabel,
	}, nil
}
