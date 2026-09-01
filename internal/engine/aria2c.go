package engine

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func FindAria2c() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "aria2c", "--version")
	if err := cmd.Run(); err == nil {
		return "aria2c", nil
	}
	return "", fmt.Errorf("aria2c not found")
}

func BuildAria2cArgs(aria2cPath string, connections int) []string {
	if aria2cPath == "" {
		return nil
	}
	c := connections
	if c < 1 {
		c = 1
	} else if c > 32 {
		c = 32
	}
	return []string{
		"--downloader", "aria2c",
		"--downloader-args", fmt.Sprintf("aria2c:-x %d -s %d -k 1M -j %d", c, c, c),
	}
}

type AriaProgress struct {
	DownloadedBytes float64
	TotalBytes      *float64
	Speed           float64
	ETA             *float64
	Connections     int
	Seeders         *int
}

var unitBytesRegex = regexp.MustCompile(`^([0-9.]+)\s*([A-Za-z]+)?$`)

func ParseUnitBytes(str string) float64 {
	match := unitBytesRegex.FindStringSubmatch(strings.TrimSpace(str))
	if match == nil {
		return 0
	}
	val, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0
	}
	unit := strings.ToLower(match[2])
	if strings.HasPrefix(unit, "g") {
		return math.Round(val * 1024 * 1024 * 1024)
	}
	if strings.HasPrefix(unit, "m") {
		return math.Round(val * 1024 * 1024)
	}
	if strings.HasPrefix(unit, "k") {
		return math.Round(val * 1024)
	}
	return math.Round(val)
}

var (
	ariaHoursRegex   = regexp.MustCompile(`(\d+)h`)
	ariaMinutesRegex = regexp.MustCompile(`(\d+)m`)
	ariaSecondsRegex = regexp.MustCompile(`(\d+)s`)
)

func ParseAriaEta(etaStr string) *float64 {
	var seconds float64
	if m := ariaHoursRegex.FindStringSubmatch(etaStr); m != nil {
		h, _ := strconv.ParseFloat(m[1], 64)
		seconds += h * 3600
	}
	if m := ariaMinutesRegex.FindStringSubmatch(etaStr); m != nil {
		min, _ := strconv.ParseFloat(m[1], 64)
		seconds += min * 60
	}
	if m := ariaSecondsRegex.FindStringSubmatch(etaStr); m != nil {
		sec, _ := strconv.ParseFloat(m[1], 64)
		seconds += sec
	}
	if seconds > 0 {
		return &seconds
	}
	return nil
}

var (
	ariaHeaderRegex = regexp.MustCompile(`^\[#[0-9a-fA-F]+\s+([0-9.]+[A-Za-z]*)/([0-9.]+[A-Za-z]*)(?:\((\d+)%\))?`)
	ariaCNRegex     = regexp.MustCompile(`\bCN:(\d+)`)
	ariaSDRegex     = regexp.MustCompile(`\bSD:(\d+)`)
	ariaDLRegex     = regexp.MustCompile(`\bDL:([0-9.]+[A-Za-z]*)`)
	ariaETARegex    = regexp.MustCompile(`\bETA:([0-9a-zA-Z]+)`)
)

func ParseAriaProgressLine(line string) *AriaProgress {
	headerMatch := ariaHeaderRegex.FindStringSubmatch(line)
	if headerMatch == nil {
		return nil
	}

	downloadedBytes := ParseUnitBytes(headerMatch[1])
	totalBytesVal := ParseUnitBytes(headerMatch[2])
	var totalBytes *float64
	if totalBytesVal > 0 {
		totalBytes = &totalBytesVal
	}

	connections := 1
	if cnMatch := ariaCNRegex.FindStringSubmatch(line); cnMatch != nil {
		if c, err := strconv.Atoi(cnMatch[1]); err == nil {
			connections = c
		}
	}

	var seeders *int
	if sdMatch := ariaSDRegex.FindStringSubmatch(line); sdMatch != nil {
		if s, err := strconv.Atoi(sdMatch[1]); err == nil {
			seeders = &s
		}
	}

	speed := float64(0)
	if dlMatch := ariaDLRegex.FindStringSubmatch(line); dlMatch != nil {
		speed = ParseUnitBytes(dlMatch[1])
	}

	var eta *float64
	if etaMatch := ariaETARegex.FindStringSubmatch(line); etaMatch != nil {
		eta = ParseAriaEta(etaMatch[1])
	} else if speed > 0 && totalBytes != nil && *totalBytes > downloadedBytes {
		calcEta := math.Round((*totalBytes - downloadedBytes) / speed)
		eta = &calcEta
	}

	return &AriaProgress{
		DownloadedBytes: downloadedBytes,
		TotalBytes:      totalBytes,
		Connections:     connections,
		Seeders:         seeders,
		Speed:           speed,
		ETA:             eta,
	}
}

type DirectDownloadOptions struct {
	Aria2cBin   string
	URL         string
	OutputDir   string
	Filename    string
	Connections int
	SpeedLimit  string
}

func BakeDirectDownload(ctx context.Context, opts DirectDownloadOptions, handlers BakeHandlers) (string, error) {
	c := opts.Connections
	if c < 1 {
		c = 16
	} else if c > 32 {
		c = 32
	}
	args := []string{
		"-d", opts.OutputDir,
		"-x", strconv.Itoa(c),
		"-s", strconv.Itoa(c),
		"-k", "1M",
		"-j", strconv.Itoa(c),
		"--connect-timeout=6",
		"--timeout=10",
		"--max-tries=2",
		"--retry-wait=1",
		"--summary-interval=1",
		"--auto-file-renaming=false",
		"--allow-overwrite=true",
	}
	if opts.SpeedLimit != "" && opts.SpeedLimit != "unlimited" {
		args = append(args, fmt.Sprintf("--max-download-limit=%s", opts.SpeedLimit))
	}
	if opts.Filename != "" {
		args = append(args, "-o", opts.Filename)
	}
	args = append(args, opts.URL)

	return runAria2Process(ctx, opts.Aria2cBin, args, opts.OutputDir, opts.Filename, handlers)
}

type TorrentDownloadOptions struct {
	Aria2cBin  string
	Target     string
	OutputDir  string
	SpeedLimit string
}

func BakeTorrentDownload(ctx context.Context, opts TorrentDownloadOptions, handlers BakeHandlers) (string, error) {
	args := []string{
		"-d", opts.OutputDir,
		"--seed-time=0",
		"--summary-interval=1",
		"--bt-stop-timeout=60",
	}
	if opts.SpeedLimit != "" && opts.SpeedLimit != "unlimited" {
		args = append(args, fmt.Sprintf("--max-download-limit=%s", opts.SpeedLimit))
	}
	args = append(args, opts.Target)
	return runAria2Process(ctx, opts.Aria2cBin, args, opts.OutputDir, "", handlers)
}

type BatchItem struct {
	URL      string   `json:"url"`
	Mirrors  []string `json:"mirrors,omitempty"`
	Filename string   `json:"filename,omitempty"`
	Name     string   `json:"name,omitempty"`
	Headers  []string `json:"headers,omitempty"`
}

type BatchDownloadOptions struct {
	Aria2cBin   string
	Items       []BatchItem
	OutputDir   string
	Connections int
	SpeedLimit  string
}

func BakeBatchDownload(ctx context.Context, opts BatchDownloadOptions, handlers BakeHandlers) (string, error) {
	c := opts.Connections
	if c < 1 {
		c = 16
	} else if c > 32 {
		c = 32
	}

	tmpFile, err := os.CreateTemp("", "anpan-batch-*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())

	var sb strings.Builder
	for _, item := range opts.Items {
		uris := item.URL
		if len(item.Mirrors) > 0 {
			uris = strings.Join(item.Mirrors, "\t")
		}
		sb.WriteString(fmt.Sprintf("%s\n", uris))
		out := item.Filename
		if out == "" {
			out = item.Name
		}
		if out != "" {
			sb.WriteString(fmt.Sprintf("  out=%s\n", out))
		}
		for _, h := range item.Headers {
			sb.WriteString(fmt.Sprintf("  header=%s\n", h))
		}
	}
	if _, err := tmpFile.WriteString(sb.String()); err != nil {
		return "", err
	}
	tmpFile.Close()

	args := []string{
		"-d", opts.OutputDir,
		"-i", tmpFile.Name(),
		"-x", strconv.Itoa(c),
		"-s", strconv.Itoa(c),
		"-k", "1M",
		"-j", strconv.Itoa(c),
		"--connect-timeout=6",
		"--timeout=10",
		"--max-tries=2",
		"--retry-wait=1",
		"--summary-interval=1",
		"--auto-file-renaming=false",
		"--allow-overwrite=true",
	}
	if opts.SpeedLimit != "" && opts.SpeedLimit != "unlimited" {
		args = append(args, fmt.Sprintf("--max-download-limit=%s", opts.SpeedLimit))
	}

	return runAria2Process(ctx, opts.Aria2cBin, args, opts.OutputDir, "", handlers)
}

var (
	ariaErrorLineRegex    = regexp.MustCompile(`^\[ERROR\]\s*(.+)$`)
	ariaNoticeFileRegex   = regexp.MustCompile(`\[NOTICE\] Download complete:\s*(.+)$`)
	ariaTableSuccessRegex = regexp.MustCompile(`^[0-9a-fA-F]+\|OK\s*\|\s*[^|]+\|(.+)$`)
)

func runAria2Process(
	ctx context.Context,
	aria2cBin string,
	args []string,
	outputDir string,
	fallbackFilename string,
	handlers BakeHandlers,
) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, aria2cBin, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	if err := cmd.Start(); err != nil {
		return "", err
	}

	var resolvedFile string
	if fallbackFilename != "" {
		resolvedFile = filepath.Join(outputDir, fallbackFilename)
	}
	var lastAriaError string

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "[ERROR]") {
				lastAriaError = line
			}
		}
	}()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if errMatch := ariaErrorLineRegex.FindStringSubmatch(line); errMatch != nil {
			lastAriaError = errMatch[1]
		}

		if aria := ParseAriaProgressLine(line); aria != nil {
			etaVal := float64(0)
			if aria.ETA != nil {
				etaVal = *aria.ETA
			}
			if handlers.OnProgress != nil {
				handlers.OnProgress(BakeProgress{
					DownloadedBytes: aria.DownloadedBytes,
					TotalBytes:      aria.TotalBytes,
					Speed:           aria.Speed,
					ETA:             etaVal,
					Part:            0,
					TotalParts:      1,
					Connections:     aria.Connections,
					Seeders:         aria.Seeders,
				})
			}
		}

		if m := ariaNoticeFileRegex.FindStringSubmatch(line); m != nil {
			resolvedFile = strings.TrimSpace(m[1])
		}
		if m := ariaTableSuccessRegex.FindStringSubmatch(line); m != nil {
			resolvedFile = strings.TrimSpace(m[1])
		}
	}

	err = cmd.Wait()
	if ctx.Err() != nil {
		return "", fmt.Errorf("download cancelled")
	}

	if err == nil {
		if resolvedFile != "" {
			return resolvedFile, nil
		}
		return outputDir, nil
	}

	exitCode := 1
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}
	return "", fmt.Errorf("%s", formatAriaError(exitCode, lastAriaError))
}

func formatAriaError(code int, rawErr string) string {
	switch code {
	case 2:
		return "Connection timed out. Storage server (n1-n4) is currently down or unresponsive."
	case 3:
		return "File not found on remote server (HTTP 404)."
	case 9:
		return "Not enough disk space available."
	case 19:
		return "DNS error: failed to resolve server hostname."
	case 24:
		return "HTTP authorization failed (HTTP 403 Forbidden)."
	}
	if strings.TrimSpace(rawErr) != "" {
		return strings.TrimSpace(rawErr)
	}
	return fmt.Sprintf("aria2c exited with code %d.", code)
}
