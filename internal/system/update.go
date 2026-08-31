package system

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	cacheTTL        = 4 * time.Hour
	githubLatestURL = "https://api.github.com/repos/KabosuNeko/anpan/releases/latest"
)

type UpdateCheckResult struct {
	UpdateAvailable bool   `json:"updateAvailable"`
	LatestVersion   string `json:"latestVersion"`
	CurrentVersion  string `json:"currentVersion"`
}

type updateCache struct {
	LastChecked   int64  `json:"lastChecked"`
	LatestVersion string `json:"latestVersion"`
}

func cacheFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "anpan", "update-cache.json")
}

func parseSemver(v string) (int, int, int) {
	clean := strings.TrimPrefix(v, "v")
	parts := strings.Split(clean, ".")
	var nums [3]int
	for i := 0; i < len(parts) && i < 3; i++ {
		n, _ := strconv.Atoi(parts[i])
		nums[i] = n
	}
	return nums[0], nums[1], nums[2]
}

func IsNewerVersion(remote, current string) bool {
	rMaj, rMin, rPatch := parseSemver(remote)
	cMaj, cMin, cPatch := parseSemver(current)

	if rMaj != cMaj {
		return rMaj > cMaj
	}
	if rMin != cMin {
		return rMin > cMin
	}
	return rPatch > cPatch
}

func readCache() *updateCache {
	data, err := os.ReadFile(cacheFilePath())
	if err != nil {
		return nil
	}
	var cache updateCache
	if err := json.Unmarshal(data, &cache); err == nil && cache.LatestVersion != "" {
		return &cache
	}
	return nil
}

func writeCache(latestVersion string) {
	p := cacheFilePath()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	c := updateCache{
		LastChecked:   time.Now().UnixMilli(),
		LatestVersion: latestVersion,
	}
	if data, err := json.MarshalIndent(c, "", "  "); err == nil {
		_ = os.WriteFile(p, append(data, '\n'), 0o644)
	}
}

type CheckUpdateOptions struct {
	Force     bool
	TimeoutMs int
}

func CheckUpdate(ctx context.Context, currentVersion string, opts *CheckUpdateOptions) *UpdateCheckResult {
	timeout := 1500 * time.Millisecond
	if opts != nil && opts.TimeoutMs > 0 {
		timeout = time.Duration(opts.TimeoutMs) * time.Millisecond
	}

	force := opts != nil && opts.Force
	now := time.Now().UnixMilli()

	if !force {
		if c := readCache(); c != nil {
			if time.Duration(now-c.LastChecked)*time.Millisecond < cacheTTL {
				return &UpdateCheckResult{
					UpdateAvailable: IsNewerVersion(c.LatestVersion, currentVersion),
					LatestVersion:   c.LatestVersion,
					CurrentVersion:  currentVersion,
				}
			}
		}
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "GET", githubLatestURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "anpan/"+currentVersion)

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		if c := readCache(); c != nil {
			return &UpdateCheckResult{
				UpdateAvailable: IsNewerVersion(c.LatestVersion, currentVersion),
				LatestVersion:   c.LatestVersion,
				CurrentVersion:  currentVersion,
			}
		}
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var ghRelease struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ghRelease); err != nil || ghRelease.TagName == "" {
		return nil
	}

	latest := strings.TrimPrefix(ghRelease.TagName, "v")
	writeCache(latest)

	return &UpdateCheckResult{
		UpdateAvailable: IsNewerVersion(latest, currentVersion),
		LatestVersion:   latest,
		CurrentVersion:  currentVersion,
	}
}
