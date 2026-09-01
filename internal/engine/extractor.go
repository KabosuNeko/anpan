package engine

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/KabosuNeko/anpan/internal/units"
)

type RawStream struct {
	FormatID       string   `json:"format_id"`
	Ext            string   `json:"ext,omitempty"`
	Vcodec         string   `json:"vcodec,omitempty"`
	Acodec         string   `json:"acodec,omitempty"`
	Height         *int     `json:"height,omitempty"`
	Width          *int     `json:"width,omitempty"`
	FPS            *float64 `json:"fps,omitempty"`
	DynamicRange   string   `json:"dynamic_range,omitempty"`
	VBR            *float64 `json:"vbr,omitempty"`
	ABR            *float64 `json:"abr,omitempty"`
	TBR            *float64 `json:"tbr,omitempty"`
	Filesize       *float64 `json:"filesize,omitempty"`
	FilesizeApprox *float64 `json:"filesize_approx,omitempty"`
}

type VideoMeta struct {
	Title        string      `json:"title"`
	Uploader     string      `json:"uploader,omitempty"`
	Duration     *float64    `json:"duration,omitempty"`
	WebpageURL   string      `json:"webpage_url,omitempty"`
	ExtractorKey string      `json:"extractor_key,omitempty"`
	Formats      []RawStream `json:"formats,omitempty"`
}

type PortionKind string

const (
	PortionKindVideo PortionKind = "video"
	PortionKindAudio PortionKind = "audio"
)

type Portion struct {
	Label     string      `json:"label"`
	Kind      PortionKind `json:"kind"`
	YtdlpArgs []string    `json:"ytdlpArgs"`
}

type ExtractPortionsOptions struct {
	EmbedMetadata  *bool
	VideoContainer string // "mp4" | "mkv" | "webm"
	VideoCodec     string // "auto" | "av1" | "vp9" | "avc"
	AudioFormat    string // "mp3" | "m4a" | "opus" | "flac" | "wav"
}

type ProbeResult struct {
	Meta           VideoMeta
	CachedJSONPath string
}

func CleanErrorOutput(stderr string) string {
	scanner := bufio.NewScanner(strings.NewReader(stderr))
	var errorLines []string
	re := regexp.MustCompile(`^ERROR:\s*(?:\[[^\]]+\]\s*)?`)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "ERROR:") {
			errorLines = append(errorLines, re.ReplaceAllString(line, ""))
		}
	}
	if len(errorLines) > 0 {
		return errorLines[len(errorLines)-1]
	}
	return ""
}

func ProbeVideo(ctx context.Context, ytdlpBin string, rawURL string) (*ProbeResult, error) {
	cmd := exec.CommandContext(ctx, ytdlpBin, "-J", "--no-playlist", "--no-warnings", rawURL)
	out, err := cmd.Output()
	if err != nil {
		stderr := ""
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = string(exitErr.Stderr)
		}
		clean := CleanErrorOutput(stderr)
		if clean != "" {
			return nil, fmt.Errorf("%s", clean)
		}
		return nil, fmt.Errorf("could not probe video: %w", err)
	}

	var meta VideoMeta
	if err := json.Unmarshal(out, &meta); err != nil {
		return nil, fmt.Errorf("could not parse media info from yt-dlp: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "anpan-meta-*.json")
	if err != nil {
		return nil, err
	}
	if _, err := tmpFile.Write(out); err != nil {
		_ = tmpFile.Close()
		return nil, err
	}
	_ = tmpFile.Close()

	return &ProbeResult{
		Meta:           meta,
		CachedJSONPath: tmpFile.Name(),
	}, nil
}

func scoreStream(s RawStream, preferredContainer string, preferredCodec string) float64 {
	score := float64(0)
	if s.TBR != nil {
		score = *s.TBR
	}
	if s.Ext == preferredContainer {
		score += 1000
	}
	switch preferredCodec {
	case "av1":
		if strings.HasPrefix(s.Vcodec, "av01") {
			score += 50000
		}
	case "vp9":
		if strings.HasPrefix(s.Vcodec, "vp09") || strings.HasPrefix(s.Vcodec, "vp9") {
			score += 50000
		}
	case "avc":
		if strings.HasPrefix(s.Vcodec, "avc") || strings.HasPrefix(s.Vcodec, "h264") {
			score += 50000
		}
	default:
		// "auto": prioritize quality-per-bitrate: AV1 > VP9 > AVC
		if strings.HasPrefix(s.Vcodec, "av01") {
			score += 3000
		} else if strings.HasPrefix(s.Vcodec, "vp09") || strings.HasPrefix(s.Vcodec, "vp9") {
			score += 2000
		} else if strings.HasPrefix(s.Vcodec, "avc") || strings.HasPrefix(s.Vcodec, "h264") {
			score += 1000
		}
	}
	return score
}

func ExtractPortions(meta VideoMeta, opts *ExtractPortionsOptions) []Portion {
	streams := meta.Formats
	var portions []Portion

	container := "mp4"
	audioFmt := "mp3"
	codec := "auto"
	embedMetadata := true
	if opts != nil {
		if opts.VideoContainer != "" {
			container = opts.VideoContainer
		}
		if opts.AudioFormat != "" {
			audioFmt = opts.AudioFormat
		}
		if opts.VideoCodec != "" {
			codec = opts.VideoCodec
		}
		if opts.EmbedMetadata != nil {
			embedMetadata = *opts.EmbedMetadata
		}
	}

	var audioStreams []RawStream
	for _, s := range streams {
		if s.Acodec != "" && s.Acodec != "none" && (s.Vcodec == "" || s.Vcodec == "none") {
			audioStreams = append(audioStreams, s)
		}
	}

	var bestAudio *RawStream
	for i := range audioStreams {
		s := &audioStreams[i]
		if bestAudio == nil {
			bestAudio = s
			continue
		}
		sRate := float64(0)
		if s.ABR != nil {
			sRate = *s.ABR
		} else if s.TBR != nil {
			sRate = *s.TBR
		}
		bestRate := float64(0)
		if bestAudio.ABR != nil {
			bestRate = *bestAudio.ABR
		} else if bestAudio.TBR != nil {
			bestRate = *bestAudio.TBR
		}
		if sRate > bestRate {
			bestAudio = s
		}
	}

	var audioSize *float64
	if bestAudio != nil {
		if bestAudio.Filesize != nil {
			audioSize = bestAudio.Filesize
		} else if bestAudio.FilesizeApprox != nil {
			audioSize = bestAudio.FilesizeApprox
		} else if (bestAudio.ABR != nil || bestAudio.TBR != nil) && meta.Duration != nil {
			rate := float64(128)
			if bestAudio.ABR != nil {
				rate = *bestAudio.ABR
			} else if bestAudio.TBR != nil {
				rate = *bestAudio.TBR
			}
			calc := math.Round(((rate * 1000) / 8) * (*meta.Duration))
			audioSize = &calc
		}
	}

	hasVideo := false
	var videoStreams []RawStream
	heightMap := make(map[int]bool)
	for _, s := range streams {
		if s.Vcodec != "" && s.Vcodec != "none" {
			hasVideo = true
			if s.Height != nil && *s.Height > 0 {
				videoStreams = append(videoStreams, s)
				heightMap[*s.Height] = true
			}
		}
	}

	var heights []int
	for h := range heightMap {
		heights = append(heights, h)
	}
	sort.Slice(heights, func(i, j int) bool {
		return heights[i] > heights[j]
	})

	for _, height := range heights {
		var candidates []RawStream
		for _, s := range videoStreams {
			if s.Height != nil && *s.Height == height {
				candidates = append(candidates, s)
			}
		}
		if len(candidates) == 0 {
			continue
		}
		sort.Slice(candidates, func(i, j int) bool {
			return scoreStream(candidates[i], container, codec) > scoreStream(candidates[j], container, codec)
		})
		best := candidates[0]
		isMuxed := best.Acodec != "" && best.Acodec != "none"

		var videoBytes *float64
		if best.Filesize != nil {
			videoBytes = best.Filesize
		} else if best.FilesizeApprox != nil {
			videoBytes = best.FilesizeApprox
		} else if (best.TBR != nil || best.VBR != nil) && meta.Duration != nil {
			rate := float64(0)
			if best.TBR != nil {
				rate = *best.TBR
			} else {
				if best.VBR != nil {
					rate += *best.VBR
				}
				if best.ABR != nil {
					rate += *best.ABR
				}
			}
			calc := math.Round(((rate * 1000) / 8) * (*meta.Duration))
			videoBytes = &calc
		}

		estimatedSize := float64(0)
		if videoBytes != nil {
			estimatedSize = *videoBytes
			if !isMuxed && audioSize != nil {
				estimatedSize += *audioSize
			}
		}

		sizeTag := ""
		if estimatedSize > 0 {
			sizeTag = fmt.Sprintf(" · ~%s", units.FormatBytes(estimatedSize))
		}
		fpsTag := ""
		if best.FPS != nil && *best.FPS >= 50 {
			fpsTag = fmt.Sprintf("%.0f", math.Round(*best.FPS))
		}
		hdrTag := ""
		if strings.Contains(strings.ToLower(best.DynamicRange), "hdr") {
			hdrTag = " HDR"
		}

		var ytdlpSelector string
		switch codec {
		case "av1":
			ytdlpSelector = fmt.Sprintf("bv*[height=%d][vcodec^=av01]+ba/bv*[height=%d]+ba/b[height=%d]/bv*[height<=%d]+ba/b", height, height, height, height)
		case "vp9":
			ytdlpSelector = fmt.Sprintf("bv*[height=%d][vcodec^=vp09]+ba/bv*[height=%d]+ba/b[height=%d]/bv*[height<=%d]+ba/b", height, height, height, height)
		case "avc":
			ytdlpSelector = fmt.Sprintf("bv*[height=%d][vcodec^=avc1]+ba/bv*[height=%d]+ba/b[height=%d]/bv*[height<=%d]+ba/b", height, height, height, height)
		default:
			ytdlpSelector = fmt.Sprintf("bv*[height=%d]+ba/b[height=%d]/bv*[height<=%d]+ba/b", height, height, height)
		}

		portions = append(portions, Portion{
			Kind:  PortionKindVideo,
			Label: fmt.Sprintf("%dp%s%s · %s%s", height, fpsTag, hdrTag, container, sizeTag),
			YtdlpArgs: []string{
				"-f",
				ytdlpSelector,
				"--merge-output-format",
				container,
			},
		})
	}

	if hasVideo && len(portions) == 0 {
		portions = append(portions, Portion{
			Kind:      PortionKindVideo,
			Label:     fmt.Sprintf("best available · %s", container),
			YtdlpArgs: []string{"-f", "bv*+ba/b", "--merge-output-format", container},
		})
	}

	addedAudioFormats := make(map[string]bool)
	nativeAudioMap := make(map[string]RawStream)

	for _, s := range audioStreams {
		codec := strings.ToLower(s.Acodec)
		ext := strings.ToLower(s.Ext)
		key := ""
		if strings.Contains(codec, "flac") || ext == "flac" {
			key = "flac"
		} else if strings.Contains(codec, "opus") || ext == "webm" {
			key = "opus"
		} else if strings.Contains(codec, "mp4a") || strings.Contains(codec, "aac") || ext == "m4a" {
			key = "m4a"
		} else if strings.Contains(codec, "mp3") || ext == "mp3" {
			key = "mp3"
		} else if strings.Contains(codec, "vorbis") || ext == "ogg" {
			key = "ogg"
		} else if ext != "" && ext != "none" {
			key = ext
		}

		if key != "" {
			existing, exists := nativeAudioMap[key]
			sRate := float64(0)
			if s.ABR != nil {
				sRate = *s.ABR
			} else if s.TBR != nil {
				sRate = *s.TBR
			}
			exRate := float64(0)
			if existing.ABR != nil {
				exRate = *existing.ABR
			} else if existing.TBR != nil {
				exRate = *existing.TBR
			}
			if !exists || sRate > exRate {
				nativeAudioMap[key] = s
			}
		}
	}

	priorityKeys := []string{"opus", "flac", "m4a", "mp3", "ogg"}
	var orderedKeys []string
	for _, pk := range priorityKeys {
		if _, ok := nativeAudioMap[pk]; ok {
			orderedKeys = append(orderedKeys, pk)
		}
	}
	for k := range nativeAudioMap {
		found := false
		for _, pk := range priorityKeys {
			if pk == k {
				found = true
				break
			}
		}
		if !found {
			orderedKeys = append(orderedKeys, k)
		}
	}

	for _, key := range orderedKeys {
		stream := nativeAudioMap[key]
		var bytes *float64
		if stream.Filesize != nil {
			bytes = stream.Filesize
		} else if stream.FilesizeApprox != nil {
			bytes = stream.FilesizeApprox
		} else if (stream.ABR != nil || stream.TBR != nil) && meta.Duration != nil {
			rate := float64(128)
			if stream.ABR != nil {
				rate = *stream.ABR
			} else if stream.TBR != nil {
				rate = *stream.TBR
			}
			calc := math.Round(((rate * 1000) / 8) * (*meta.Duration))
			bytes = &calc
		}

		sizeTag := ""
		if bytes != nil {
			sizeTag = fmt.Sprintf(" · ~%s", units.FormatBytes(*bytes))
		}
		abrTag := ""
		if stream.ABR != nil {
			abrTag = fmt.Sprintf(" · ~%.0fkbps", math.Round(*stream.ABR))
		}
		isLossless := key == "flac" || key == "wav"
		tagSuffix := " (original)"
		if isLossless {
			tagSuffix = " (lossless)"
		} else if key == "m4a" {
			tagSuffix = " (aac)"
		}

		ytdlpSelector := fmt.Sprintf("ba[ext=%s]/ba", stream.Ext)
		if key == "opus" {
			ytdlpSelector = "ba[acodec^=opus]/ba"
		} else if key == "m4a" {
			ytdlpSelector = "ba[ext=m4a]/ba"
		} else if stream.FormatID != "" {
			ytdlpSelector = fmt.Sprintf("%s/ba", stream.FormatID)
		}

		audioArgs := []string{"-f", ytdlpSelector, "-x", "--audio-format", key}
		if embedMetadata {
			if key != "wav" {
				audioArgs = append(audioArgs, "--embed-thumbnail")
			}
			audioArgs = append(audioArgs, "--add-metadata")
		}

		portions = append(portions, Portion{
			Kind:      PortionKindAudio,
			Label:     fmt.Sprintf("audio only · %s%s%s%s", key, tagSuffix, abrTag, sizeTag),
			YtdlpArgs: audioArgs,
		})
		addedAudioFormats[key] = true
	}

	if !addedAudioFormats[audioFmt] {
		audioSizeTag := ""
		if audioSize != nil {
			audioSizeTag = fmt.Sprintf(" · ~%s", units.FormatBytes(*audioSize))
		}
		audioBitrateTag := ""
		if bestAudio != nil && bestAudio.ABR != nil {
			audioBitrateTag = fmt.Sprintf(" · %.0fkbps", math.Round(*bestAudio.ABR))
		}
		audioArgs := []string{"-f", "ba/b", "-x", "--audio-format", audioFmt, "--audio-quality", "0"}
		if embedMetadata {
			if audioFmt != "wav" {
				audioArgs = append(audioArgs, "--embed-thumbnail")
			}
			audioArgs = append(audioArgs, "--add-metadata")
		}
		portions = append(portions, Portion{
			Kind:      PortionKindAudio,
			Label:     fmt.Sprintf("audio only · %s%s%s", audioFmt, audioBitrateTag, audioSizeTag),
			YtdlpArgs: audioArgs,
		})
		addedAudioFormats[audioFmt] = true
	}

	if !addedAudioFormats["mp3"] {
		mp3SizeTag := ""
		if audioSize != nil {
			mp3SizeTag = fmt.Sprintf(" · ~%s", units.FormatBytes(*audioSize))
		}
		mp3Args := []string{"-f", "ba/b", "-x", "--audio-format", "mp3", "--audio-quality", "0"}
		if embedMetadata {
			mp3Args = append(mp3Args, "--embed-thumbnail", "--add-metadata")
		}
		portions = append(portions, Portion{
			Kind:      PortionKindAudio,
			Label:     fmt.Sprintf("audio only · mp3 · 320kbps%s", mp3SizeTag),
			YtdlpArgs: mp3Args,
		})
		addedAudioFormats["mp3"] = true
	}

	if len(portions) == 0 {
		portions = append(portions, Portion{
			Kind:      PortionKindVideo,
			Label:     fmt.Sprintf("best available · %s", container),
			YtdlpArgs: []string{"-f", "bv*+ba/b", "--merge-output-format", container},
		})
	}

	return portions
}

var albumPrefixRegex = regexp.MustCompile(`(?i)^(?:album|ep|single)\s*[-:–—]\s*`)

func CleanAlbumTitle(title string) string {
	if strings.TrimSpace(title) == "" {
		return "Playlist"
	}
	cleaned := albumPrefixRegex.ReplaceAllString(strings.TrimSpace(title), "")
	if strings.TrimSpace(cleaned) == "" {
		return title
	}
	return strings.TrimSpace(cleaned)
}

type PlaylistMeta struct {
	Title      string `json:"title"`
	Uploader   string `json:"uploader,omitempty"`
	TrackCount int    `json:"trackCount"`
	WebpageURL string `json:"webpage_url"`
}

func ProbePlaylist(ctx context.Context, ytdlpBin string, rawURL string) (*PlaylistMeta, error) {
	cmd := exec.CommandContext(ctx, ytdlpBin, "--flat-playlist", "-J", "--no-warnings", rawURL)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var data struct {
		Type          string `json:"_type"`
		Title         string `json:"title"`
		Uploader      string `json:"uploader"`
		PlaylistCount *int   `json:"playlist_count"`
		Entries       []any  `json:"entries"`
		WebpageURL    string `json:"webpage_url"`
	}

	if err := json.Unmarshal(out, &data); err != nil {
		return nil, err
	}

	if data.Type == "playlist" || len(data.Entries) > 0 {
		count := len(data.Entries)
		if data.PlaylistCount != nil {
			count = *data.PlaylistCount
		}
		webURL := data.WebpageURL
		if webURL == "" {
			webURL = rawURL
		}
		return &PlaylistMeta{
			Title:      CleanAlbumTitle(data.Title),
			Uploader:   data.Uploader,
			TrackCount: count,
			WebpageURL: webURL,
		}, nil
	}

	return nil, fmt.Errorf("not a playlist")
}

func ExtractPlaylistPortions(opts *ExtractPortionsOptions) []Portion {
	container := "mp4"
	audioFmt := "mp3"
	codec := "auto"
	embedMetadata := true
	if opts != nil {
		if opts.VideoContainer != "" {
			container = opts.VideoContainer
		}
		if opts.AudioFormat != "" {
			audioFmt = opts.AudioFormat
		}
		if opts.VideoCodec != "" {
			codec = opts.VideoCodec
		}
		if opts.EmbedMetadata != nil {
			embedMetadata = *opts.EmbedMetadata
		}
	}

	audioArgs := []string{"-f", "ba/b", "-x", "--audio-format", audioFmt, "--audio-quality", "0"}
	if embedMetadata {
		audioArgs = append(audioArgs, "--embed-thumbnail", "--add-metadata")
	}

	portions := []Portion{
		{
			Kind:      PortionKindAudio,
			Label:     fmt.Sprintf("all tracks · %s (audio only)", audioFmt),
			YtdlpArgs: audioArgs,
		},
	}

	if audioFmt != "opus" {
		opusArgs := []string{"-f", "ba[acodec^=opus]/ba", "-x", "--audio-format", "opus"}
		if embedMetadata {
			opusArgs = append(opusArgs, "--embed-thumbnail", "--add-metadata")
		}
		portions = append(portions, Portion{
			Kind:      PortionKindAudio,
			Label:     "all tracks · opus (original audio)",
			YtdlpArgs: opusArgs,
		})
	}

	if audioFmt != "mp3" {
		mp3Args := []string{"-f", "ba/b", "-x", "--audio-format", "mp3", "--audio-quality", "0"}
		if embedMetadata {
			mp3Args = append(mp3Args, "--embed-thumbnail", "--add-metadata")
		}
		portions = append(portions, Portion{
			Kind:      PortionKindAudio,
			Label:     "all tracks · mp3 (audio only)",
			YtdlpArgs: mp3Args,
		})
	}

	var videoSelector string
	switch codec {
	case "av1":
		videoSelector = "bv*[vcodec^=av01]+ba/bv*+ba/b"
	case "vp9":
		videoSelector = "bv*[vcodec^=vp09]+ba/bv*+ba/b"
	case "avc":
		videoSelector = "bv*[vcodec^=avc1]+ba/bv*+ba/b"
	default:
		videoSelector = "bv*+ba/b"
	}

	portions = append(portions, Portion{
		Kind:      PortionKindVideo,
		Label:     fmt.Sprintf("all videos · %s (best quality)", container),
		YtdlpArgs: []string{"-f", videoSelector, "--merge-output-format", container},
	})

	return portions
}
