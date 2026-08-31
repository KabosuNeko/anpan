package engine

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const releaseBase = "https://github.com/yt-dlp/yt-dlp/releases/latest/download"

func AnpanBinDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".anpan", "bin")
}

func ytDlpAssetName() string {
	if runtime.GOOS == "windows" {
		return "yt-dlp.exe"
	}
	if runtime.GOOS == "darwin" {
		return "yt-dlp_macos"
	}
	if runtime.GOARCH == "arm64" {
		return "yt-dlp_linux_aarch64"
	}
	return "yt-dlp_linux"
}

func BinaryResponds(cmd string, args ...string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, cmd, args...)
	return c.Run() == nil
}

func HasMutagen(ytdlpPath string) bool {
	if strings.Contains(ytdlpPath, ".anpan") || strings.Contains(ytdlpPath, "yt-dlp_") {
		return true
	}
	binName := "yt-dlp"
	if runtime.GOOS == "windows" {
		binName = "yt-dlp.exe"
	}
	localBin := filepath.Join(AnpanBinDir(), binName)
	if BinaryResponds(localBin, "--version") {
		return true
	}
	return BinaryResponds("python3", "-c", "import mutagen")
}

func safeReplaceBinary(src, dst string) error {
	for attempt := 0; attempt < 5; attempt++ {
		_ = os.Remove(dst)
		err := os.Rename(src, dst)
		if err == nil {
			return nil
		}
		if attempt == 4 {
			// fallback copy
			data, readErr := os.ReadFile(src)
			if readErr != nil {
				return readErr
			}
			if writeErr := os.WriteFile(dst, data, 0o755); writeErr != nil {
				return writeErr
			}
			_ = os.Remove(src)
			return nil
		}
		time.Sleep(time.Duration(200*(attempt+1)) * time.Millisecond)
	}
	return nil
}

func EnsureYtDlpBinary(ctx context.Context, onStatus func(msg string)) (string, error) {
	binName := "yt-dlp"
	if runtime.GOOS == "windows" {
		binName = "yt-dlp.exe"
	}
	localBin := filepath.Join(AnpanBinDir(), binName)

	if BinaryResponds(localBin, "--version") {
		return localBin, nil
	}

	hasSystemYtDlp := BinaryResponds("yt-dlp", "--version")
	hasSysMutagen := BinaryResponds("python3", "-c", "import mutagen")
	if hasSystemYtDlp && hasSysMutagen {
		return "yt-dlp", nil
	}

	if onStatus != nil {
		onStatus("fetching self-contained yt-dlp (bundled dependencies)…")
	}

	if err := os.MkdirAll(AnpanBinDir(), 0o755); err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/%s", releaseBase, ytDlpAssetName())
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		if hasSystemYtDlp {
			return "yt-dlp", nil
		}
		return "", err
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		if hasSystemYtDlp {
			return "yt-dlp", nil
		}
		return "", fmt.Errorf("could not download standalone yt-dlp: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if hasSystemYtDlp {
			return "yt-dlp", nil
		}
		return "", fmt.Errorf("download yt-dlp failed: %s", resp.Status)
	}

	tmpPath := localBin + ".download"
	out, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(out, resp.Body)
	_ = out.Close()
	if err != nil {
		_ = os.Remove(tmpPath)
		if hasSystemYtDlp {
			return "yt-dlp", nil
		}
		return "", err
	}

	if err := safeReplaceBinary(tmpPath, localBin); err != nil {
		if hasSystemYtDlp {
			return "yt-dlp", nil
		}
		return "", err
	}
	_ = os.Chmod(localBin, 0o755)

	return localBin, nil
}
