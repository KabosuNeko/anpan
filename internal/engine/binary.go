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
	"sync"
	"time"

	"github.com/KabosuNeko/anpan/internal/units"
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

// localYtDlp returns the path of the anpan-managed yt-dlp binary.
func localYtDlp() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(AnpanBinDir(), "yt-dlp.exe")
	}
	return filepath.Join(AnpanBinDir(), "yt-dlp")
}

func HasMutagen(ytdlpPath string) bool {
	if strings.Contains(ytdlpPath, ".anpan") || strings.Contains(ytdlpPath, "yt-dlp_") {
		return true
	}
	if BinaryResponds(localYtDlp(), "--version") {
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

// ytDlpUpdateAge is how old the managed yt-dlp may get before anpan runs its self-update.
const ytDlpUpdateAge = 7 * 24 * time.Hour

var resolvedYtDlp struct {
	sync.Mutex
	path string
}

// EnsureYtDlpBinary returns a usable yt-dlp, downloading or refreshing the managed copy as needed.
// A successful result is cached for the life of the process.
func EnsureYtDlpBinary(ctx context.Context, onStatus func(msg string)) (string, error) {
	resolvedYtDlp.Lock()
	defer resolvedYtDlp.Unlock()
	if resolvedYtDlp.path != "" {
		return resolvedYtDlp.path, nil
	}
	path, err := ensureYtDlpBinary(ctx, onStatus)
	if err == nil {
		resolvedYtDlp.path = path
	}
	return path, err
}

// refreshYtDlp self-updates the managed yt-dlp at most once per ytDlpUpdateAge; failures keep the current binary.
func refreshYtDlp(ctx context.Context, localBin string, onStatus func(msg string)) {
	fi, err := os.Stat(localBin)
	if err != nil || time.Since(fi.ModTime()) < ytDlpUpdateAge {
		return
	}
	if onStatus != nil {
		onStatus("updating yt-dlp…")
	}
	updCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	_ = exec.CommandContext(updCtx, localBin, "-U").Run()
	now := time.Now()
	_ = os.Chtimes(localBin, now, now)
}

func ensureYtDlpBinary(ctx context.Context, onStatus func(msg string)) (string, error) {
	localBin := localYtDlp()

	if BinaryResponds(localBin, "--version") {
		refreshYtDlp(ctx, localBin, onStatus)
		return localBin, nil
	}

	hasSystemYtDlp := BinaryResponds("yt-dlp", "--version")
	hasSysMutagen := BinaryResponds("python3", "-c", "import mutagen")
	if hasSystemYtDlp && hasSysMutagen {
		return "yt-dlp", nil
	}

	fallback := func(err error) (string, error) {
		if hasSystemYtDlp {
			return "yt-dlp", nil
		}
		return "", err
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
		return fallback(err)
	}

	// No client timeout: the binary is ~35 MB and slow links need longer; ctx lets the user cancel.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fallback(fmt.Errorf("could not download standalone yt-dlp: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fallback(fmt.Errorf("download yt-dlp failed: %s", resp.Status))
	}

	tmpPath := localBin + ".download"
	out, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return "", err
	}

	var body io.Reader = resp.Body
	if onStatus != nil {
		body = &progressReader{r: resp.Body, total: resp.ContentLength, onStatus: onStatus}
	}
	_, err = io.Copy(out, body)
	_ = out.Close()
	if err != nil {
		_ = os.Remove(tmpPath)
		return fallback(err)
	}

	if err := safeReplaceBinary(tmpPath, localBin); err != nil {
		return fallback(err)
	}
	_ = os.Chmod(localBin, 0o755)

	return localBin, nil
}

// progressReader reports download progress through onStatus at most twice a second.
type progressReader struct {
	r        io.Reader
	read     int64
	total    int64
	last     time.Time
	onStatus func(msg string)
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.read += int64(n)
	if time.Since(p.last) >= 500*time.Millisecond {
		p.last = time.Now()
		msg := "downloading yt-dlp " + units.FormatBytes(float64(p.read))
		if p.total > 0 {
			msg += " / " + units.FormatBytes(float64(p.total))
		}
		p.onStatus(msg + "…")
	}
	return n, err
}
