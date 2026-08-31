package engine

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const progressTag = "ANPAN|"
const progressTemplate = progressTag + "%(progress.downloaded_bytes)s|%(progress.total_bytes)s|%(progress.total_bytes_estimate)s|%(progress.speed)s|%(progress.eta)s"

type BakeVideoOptions struct {
	YtdlpBin       string
	FfmpegLocation string
	Aria2cArgs     []string
	URL            string
	CachedJSONPath string
	Portion        Portion
	OutputDir      string
	TimeRange      string
	IsPlaylist     bool
	CookiesBrowser string
	Subtitles      string // "off" | "embed" | "write"
	SubLangs       string
	SponsorBlock   string // "off" | "remove" | "mark"
	WriteThumbnail bool
}

func parseProgressFloat(v string) *float64 {
	clean := strings.TrimSpace(v)
	if clean == "" || clean == "NA" || clean == "None" {
		return nil
	}
	if f, err := strconv.ParseFloat(clean, 64); err == nil {
		return &f
	}
	return nil
}

var (
	playlistItemRegex    = regexp.MustCompile(`^\[download\] Downloading item (\d+) of (\d+)`)
	formatCountRegex     = regexp.MustCompile(`Downloading 1 format\(s\):\s*(.+)`)
	mergerTargetRegex    = regexp.MustCompile(`^\[Merger\] Merging formats into "(.+)"$`)
	extractAudioTargetRegex = regexp.MustCompile(`^\[ExtractAudio\] Destination: (.+)$`)
)

func BakeVideo(ctx context.Context, opts BakeVideoOptions, handlers BakeHandlers) (string, error) {
	var outputTemplate string
	if opts.IsPlaylist {
		outputTemplate = filepath.Join(opts.OutputDir, "%(playlist_title,playlist)s", "%(playlist_index)02d - %(title)s.%(ext)s")
	} else {
		trimSuffix := ""
		if opts.TimeRange != "" {
			cleanRange := strings.ReplaceAll(strings.ReplaceAll(opts.TimeRange, ":", "."), "*", ".")
			trimSuffix = fmt.Sprintf(" [%s]", cleanRange)
		}
		outputTemplate = filepath.Join(opts.OutputDir, fmt.Sprintf("%%(title)s%s.%%(ext)s", trimSuffix))
	}

	portionArgs := opts.Portion.YtdlpArgs
	isOpusOrOgg := strings.Contains(strings.ToLower(opts.Portion.Label), "opus") || strings.Contains(strings.ToLower(opts.Portion.Label), "ogg")
	if isOpusOrOgg && !HasMutagen(opts.YtdlpBin) {
		var filtered []string
		for _, a := range portionArgs {
			if a != "--embed-thumbnail" {
				filtered = append(filtered, a)
			}
		}
		portionArgs = filtered
	}

	var args []string
	if opts.CachedJSONPath != "" && !opts.IsPlaylist {
		args = append(args, "--load-info-json", opts.CachedJSONPath)
	} else {
		args = append(args, opts.URL)
	}
	args = append(args, portionArgs...)

	if opts.IsPlaylist {
		args = append(args,
			"--yes-playlist",
			"--ignore-errors",
			"--replace-in-metadata", "playlist_title", "(?i)^(?:album|ep|single)\\s*[-:–—]\\s*", "",
			"--replace-in-metadata", "playlist", "(?i)^(?:album|ep|single)\\s*[-:–—]\\s*", "",
		)
	} else {
		args = append(args, "--no-playlist")
	}

	args = append(args,
		"--no-warnings",
		"--newline",
		"--no-quiet",
		"--progress",
		"--progress-template", "download:"+progressTemplate,
		"--print", "after_move:filepath",
		"--no-simulate",
		"-o", outputTemplate,
	)

	if opts.TimeRange != "" {
		args = append(args, "--download-sections", "*"+opts.TimeRange)
	}
	if opts.FfmpegLocation != "" {
		args = append(args, "--ffmpeg-location", opts.FfmpegLocation)
	}
	if len(opts.Aria2cArgs) > 0 {
		args = append(args, opts.Aria2cArgs...)
	}
	if opts.CookiesBrowser != "" && opts.CookiesBrowser != "none" {
		args = append(args, "--cookies-from-browser", opts.CookiesBrowser)
	}
	if opts.SponsorBlock == "remove" {
		args = append(args, "--sponsorblock-remove", "all")
	} else if opts.SponsorBlock == "mark" {
		args = append(args, "--sponsorblock-mark", "all")
	}
	if opts.Subtitles == "embed" {
		subLangs := "vi,en"
		if opts.SubLangs != "" {
			subLangs = opts.SubLangs
		}
		args = append(args, "--embed-subs", "--sub-langs", subLangs)
	} else if opts.Subtitles == "write" {
		subLangs := "vi,en"
		if opts.SubLangs != "" {
			subLangs = opts.SubLangs
		}
		args = append(args, "--write-subs", "--sub-langs", subLangs)
	}
	if opts.WriteThumbnail {
		args = append(args, "--write-thumbnail")
	}

	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, opts.YtdlpBin, args...)
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

	var finalFilepath string
	var destinations []string
	var lastStderr string
	part := 0
	totalParts := 1
	lastDownloaded := float64(0)
	var playlistItem, playlistTotal int

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "ERROR:") {
				lastStderr = line
			}
		}
	}()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if m := playlistItemRegex.FindStringSubmatch(line); m != nil {
			playlistItem, _ = strconv.Atoi(m[1])
			playlistTotal, _ = strconv.Atoi(m[2])
		}

		if strings.HasPrefix(line, progressTag) {
			body := strings.TrimPrefix(line, progressTag)
			parts := strings.Split(body, "|")
			if len(parts) >= 5 {
				dlVal := parseProgressFloat(parts[0])
				downloaded := float64(0)
				if dlVal != nil {
					downloaded = *dlVal
				}
				if downloaded < lastDownloaded {
					part++
				}
				lastDownloaded = downloaded

				total := parseProgressFloat(parts[1])
				if total == nil {
					total = parseProgressFloat(parts[2])
				}
				speed := float64(0)
				if s := parseProgressFloat(parts[3]); s != nil {
					speed = *s
				}
				eta := float64(0)
				if e := parseProgressFloat(parts[4]); e != nil {
					eta = *e
				}

				if handlers.OnProgress != nil {
					handlers.OnProgress(BakeProgress{
						DownloadedBytes: downloaded,
						TotalBytes:      total,
						Speed:           speed,
						ETA:             eta,
						Part:            part,
						TotalParts:      totalParts,
						PlaylistItem:    playlistItem,
						PlaylistTotal:   playlistTotal,
					})
				}
			}
		} else if aria := ParseAriaProgressLine(line); aria != nil {
			if aria.DownloadedBytes < lastDownloaded && aria.DownloadedBytes > 0 {
				part++
			}
			lastDownloaded = aria.DownloadedBytes

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
					Part:            part,
					TotalParts:      totalParts,
					PlaylistItem:    playlistItem,
					PlaylistTotal:   playlistTotal,
					Connections:     aria.Connections,
					Seeders:         aria.Seeders,
				})
			}
		}

		if m := formatCountRegex.FindStringSubmatch(line); m != nil {
			totalParts = len(strings.Split(m[1], "+"))
		} else if strings.Contains(line, "[Merger]") || strings.Contains(line, "[ExtractAudio]") {
			if m := mergerTargetRegex.FindStringSubmatch(line); m != nil {
				destinations = append(destinations, m[1])
			} else if m := extractAudioTargetRegex.FindStringSubmatch(line); m != nil {
				destinations = append(destinations, m[1])
			}
			if handlers.OnProcessing != nil {
				handlers.OnProcessing()
			}
		} else if strings.HasPrefix(line, "[download] Destination: ") {
			dest := strings.TrimPrefix(line, "[download] Destination: ")
			destinations = append(destinations, dest)
		} else if filepath.IsAbs(line) {
			finalFilepath = line
		}
	}

	err = cmd.Wait()
	if ctx.Err() != nil {
		for _, d := range destinations {
			_ = os.Remove(d)
			_ = os.Remove(d + ".part")
			_ = os.Remove(d + ".ytdl")
		}
		return "", fmt.Errorf("download cancelled")
	}

	if err != nil {
		clean := CleanErrorOutput(lastStderr)
		if clean != "" {
			return "", fmt.Errorf("%s", clean)
		}
		return "", fmt.Errorf("download failed: %w", err)
	}

	if finalFilepath != "" {
		if opts.IsPlaylist {
			return filepath.Dir(finalFilepath), nil
		}
		return finalFilepath, nil
	}

	return opts.OutputDir, nil
}
