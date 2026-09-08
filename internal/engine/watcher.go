package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WatcherConfig specifies configuration for the directory watcher.
type WatcherConfig struct {
	WatchDir       string
	OutDir         string
	CheckInterval  time.Duration
	Aria2cLimit    string
	Connections    int
	OnDownloadDone func(file string, target string)
	OnError        func(file string, err error)
}

// StartWatcher periodically scans the watch directory for .torrent, .magnet, or link files and triggers downloads.
func StartWatcher(ctx context.Context, cfg WatcherConfig) error {
	if cfg.WatchDir == "" {
		return fmt.Errorf("watch directory must not be empty")
	}
	if cfg.CheckInterval <= 0 {
		cfg.CheckInterval = 3 * time.Second
	}

	if err := os.MkdirAll(cfg.WatchDir, 0755); err != nil {
		return fmt.Errorf("create watch directory: %w", err)
	}

	processedDir := filepath.Join(cfg.WatchDir, ".processed")
	if err := os.MkdirAll(processedDir, 0755); err != nil {
		return fmt.Errorf("create processed directory: %w", err)
	}

	ticker := time.NewTicker(cfg.CheckInterval)
	defer ticker.Stop()

	// Initial scan
	_ = scanWatchDir(ctx, cfg, processedDir)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			_ = scanWatchDir(ctx, cfg, processedDir)
		}
	}
}

func scanWatchDir(ctx context.Context, cfg WatcherConfig, processedDir string) error {
	entries, err := os.ReadDir(cfg.WatchDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}

		filePath := filepath.Join(cfg.WatchDir, name)
		if !IsWatchCandidate(name) {
			continue
		}

		targetURI, err := extractTargetFromWatchFile(filePath)
		if err != nil || targetURI == "" {
			if cfg.OnError != nil {
				cfg.OnError(name, fmt.Errorf("extract target: %w", err))
			}
			continue
		}

		// Download candidate
		dest := cfg.OutDir
		if dest == "" {
			dest = cfg.WatchDir
		}

		ariaBin, err := FindAria2c()
		if err != nil {
			if cfg.OnError != nil {
				cfg.OnError(name, fmt.Errorf("aria2c missing: %w", err))
			}
			continue
		}

		// Download with aria2c
		var bakeErr error
		if strings.HasPrefix(strings.ToLower(targetURI), "magnet:") || strings.HasSuffix(strings.ToLower(targetURI), ".torrent") {
			_, bakeErr = BakeTorrentDownload(ctx, TorrentDownloadOptions{
				Aria2cBin:  ariaBin,
				Target:     targetURI,
				OutputDir:  dest,
				SpeedLimit: cfg.Aria2cLimit,
			}, BakeHandlers{})
		} else {
			conns := cfg.Connections
			if conns <= 0 {
				conns = 8
			}
			_, bakeErr = BakeDirectDownload(ctx, DirectDownloadOptions{
				Aria2cBin:   ariaBin,
				URL:         targetURI,
				OutputDir:   dest,
				Connections: conns,
				SpeedLimit:  cfg.Aria2cLimit,
			}, BakeHandlers{})
		}
		if bakeErr != nil {
			if cfg.OnError != nil {
				cfg.OnError(name, bakeErr)
			}
			continue
		}

		// Move to .processed
		destProcessed := filepath.Join(processedDir, fmt.Sprintf("%d_%s", time.Now().Unix(), name))
		_ = os.Rename(filePath, destProcessed)

		if cfg.OnDownloadDone != nil {
			cfg.OnDownloadDone(name, targetURI)
		}
	}
	return nil
}

// IsWatchCandidate checks if a file extension is supported for auto-downloading.
func IsWatchCandidate(filename string) bool {
	lower := strings.ToLower(filename)
	return strings.HasSuffix(lower, ".torrent") ||
		strings.HasSuffix(lower, ".magnet") ||
		strings.HasSuffix(lower, ".txt") ||
		strings.HasSuffix(lower, ".link")
}

func extractTargetFromWatchFile(path string) (string, error) {
	if strings.HasSuffix(strings.ToLower(path), ".torrent") {
		return path, nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		return trimmed, nil
	}
	return "", fmt.Errorf("no valid URL or magnet found in file")
}
