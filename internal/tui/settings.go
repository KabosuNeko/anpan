package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
	"github.com/KabosuNeko/anpan/internal/system"
	"github.com/KabosuNeko/anpan/internal/units"
	"github.com/mattn/go-runewidth"
)

type SettingItem struct {
	Key   string
	Label string
}

var SettingItems = []SettingItem{
	{Key: "askSaveDir", Label: "ask save location"},
	{Key: "outDir", Label: "default save dir"},
	{Key: "videoContainer", Label: "video format (container)"},
	{Key: "videoCodec", Label: "video codec preference"},
	{Key: "audioFormat", Label: "audio format"},
	{Key: "subtitles", Label: "subtitles"},
	{Key: "subLangs", Label: "subtitle languages"},
	{Key: "sponsorBlock", Label: "sponsorblock"},
	{Key: "cookiesBrowser", Label: "browser cookies"},
	{Key: "aria2c", Label: "aria2c accelerator"},
	{Key: "connections", Label: "aria2c connections (-x -s)"},
	{Key: "embedMetadata", Label: "embed audio tags & cover"},
	{Key: "writeThumbnail", Label: "write thumbnail image"},
	{Key: "lyrics", Label: "download synced lyrics (.lrc)"},
	{Key: "speedLimit", Label: "speed limit"},
	{Key: "notifications", Label: "desktop notifications"},
	{Key: "preferQuality", Label: "auto-select quality"},
}

var (
	connectionChoices = []int{4, 8, 16, 32}
	qualityChoices    = []string{"ask", "best", "1080p", "audio"}
	containerChoices  = []string{"mp4", "mkv", "webm"}
	codecChoices      = []string{"auto", "av1", "vp9", "avc"}
	audioChoices      = []string{"mp3", "m4a", "opus", "flac", "wav"}
	lyricsChoices     = []string{"synced", "off"}
	speedLimitChoices = []string{"unlimited", "1M", "5M", "10M", "20M", "50M"}
	subtitleChoices   = []string{"off", "embed", "write"}
	sublangChoices    = []string{"vi,en", "all", "en"}
	sponsorChoices    = []string{"off", "remove", "mark"}
	cookiesChoices    = []string{"none", "chrome", "firefox", "brave", "edge", "safari"}
)

func getDirPresets() []string {
	home, _ := os.UserHomeDir()
	cwd, _ := os.Getwd()
	return []string{
		filepath.Join(home, "Downloads"),
		filepath.Join(home, "Videos"),
		filepath.Join(home, "Desktop"),
		cwd,
	}
}

func FormatSettingVal(key string, cfg system.AnpanConfig) string {
	home, _ := os.UserHomeDir()
	switch key {
	case "aria2c":
		if cfg.Aria2c {
			return "on"
		}
		return "off"
	case "askSaveDir":
		if cfg.AskSaveDir {
			return "always ask"
		}
		return "use default"
	case "embedMetadata":
		if cfg.EmbedMetadata {
			return "on"
		}
		return "off"
	case "writeThumbnail":
		if cfg.WriteThumbnail {
			return "on"
		}
		return "off"
	case "notifications":
		if cfg.Notifications {
			return "on"
		}
		return "off"
	case "speedLimit":
		if cfg.SpeedLimit == "" {
			return "unlimited"
		}
		return cfg.SpeedLimit
	case "lyrics":
		if cfg.Lyrics == "" {
			return "synced"
		}
		return cfg.Lyrics
	case "connections":
		return fmt.Sprintf("%d", cfg.Connections)
	case "videoContainer":
		return cfg.VideoContainer
	case "videoCodec":
		return cfg.VideoCodec
	case "audioFormat":
		return cfg.AudioFormat
	case "subtitles":
		return cfg.Subtitles
	case "subLangs":
		return cfg.SubLangs
	case "sponsorBlock":
		return cfg.SponsorBlock
	case "cookiesBrowser":
		return cfg.CookiesBrowser
	case "preferQuality":
		return cfg.PreferQuality
	case "outDir":
		return units.ShortenPath(cfg.OutDir, home, 22)
	default:
		return ""
	}
}

func CycleConfig(cfg *system.AnpanConfig, key string, dir int) {
	switch key {
	case "aria2c":
		cfg.Aria2c = !cfg.Aria2c
	case "askSaveDir":
		cfg.AskSaveDir = !cfg.AskSaveDir
	case "embedMetadata":
		cfg.EmbedMetadata = !cfg.EmbedMetadata
	case "writeThumbnail":
		cfg.WriteThumbnail = !cfg.WriteThumbnail
	case "notifications":
		cfg.Notifications = !cfg.Notifications
	case "connections":
		idx := 2
		for i, c := range connectionChoices {
			if c == cfg.Connections {
				idx = i
				break
			}
		}
		next := (idx + dir + len(connectionChoices)) % len(connectionChoices)
		cfg.Connections = connectionChoices[next]
	case "preferQuality":
		idx := 0
		for i, q := range qualityChoices {
			if q == cfg.PreferQuality {
				idx = i
				break
			}
		}
		next := (idx + dir + len(qualityChoices)) % len(qualityChoices)
		cfg.PreferQuality = qualityChoices[next]
	case "videoContainer":
		idx := 0
		for i, c := range containerChoices {
			if c == cfg.VideoContainer {
				idx = i
				break
			}
		}
		next := (idx + dir + len(containerChoices)) % len(containerChoices)
		cfg.VideoContainer = containerChoices[next]
	case "videoCodec":
		idx := 0
		for i, c := range codecChoices {
			if c == cfg.VideoCodec {
				idx = i
				break
			}
		}
		next := (idx + dir + len(codecChoices)) % len(codecChoices)
		cfg.VideoCodec = codecChoices[next]
	case "audioFormat":
		idx := 0
		for i, a := range audioChoices {
			if a == cfg.AudioFormat {
				idx = i
				break
			}
		}
		next := (idx + dir + len(audioChoices)) % len(audioChoices)
		cfg.AudioFormat = audioChoices[next]
	case "subtitles":
		idx := 0
		for i, s := range subtitleChoices {
			if s == cfg.Subtitles {
				idx = i
				break
			}
		}
		next := (idx + dir + len(subtitleChoices)) % len(subtitleChoices)
		cfg.Subtitles = subtitleChoices[next]
	case "subLangs":
		idx := 0
		for i, l := range sublangChoices {
			if l == cfg.SubLangs {
				idx = i
				break
			}
		}
		next := (idx + dir + len(sublangChoices)) % len(sublangChoices)
		cfg.SubLangs = sublangChoices[next]
	case "sponsorBlock":
		idx := 0
		for i, s := range sponsorChoices {
			if s == cfg.SponsorBlock {
				idx = i
				break
			}
		}
		next := (idx + dir + len(sponsorChoices)) % len(sponsorChoices)
		cfg.SponsorBlock = sponsorChoices[next]
	case "cookiesBrowser":
		idx := 0
		for i, b := range cookiesChoices {
			if b == cfg.CookiesBrowser {
				idx = i
				break
			}
		}
		next := (idx + dir + len(cookiesChoices)) % len(cookiesChoices)
		cfg.CookiesBrowser = cookiesChoices[next]
	case "speedLimit":
		idx := 0
		for i, s := range speedLimitChoices {
			if s == cfg.SpeedLimit {
				idx = i
				break
			}
		}
		next := (idx + dir + len(speedLimitChoices)) % len(speedLimitChoices)
		cfg.SpeedLimit = speedLimitChoices[next]
	case "lyrics":
		idx := 0
		for i, l := range lyricsChoices {
			if l == cfg.Lyrics {
				idx = i
				break
			}
		}
		next := (idx + dir + len(lyricsChoices)) % len(lyricsChoices)
		cfg.Lyrics = lyricsChoices[next]
	case "outDir":
		presets := getDirPresets()
		curNorm := filepath.Clean(cfg.OutDir)
		idx := -1
		for i, p := range presets {
			if filepath.Clean(p) == curNorm {
				idx = i
				break
			}
		}
		if idx == -1 {
			idx = 0
		}
		next := (idx + dir + len(presets)) % len(presets)
		cfg.OutDir = presets[next]
	}
}

func RenderSettingsView(width int, selectedIndex int, editingDir bool, dirInput textinput.Model, cfg system.AnpanConfig) string {
	rowWidth := width - 4
	var lines []string

	for i, item := range SettingItems {
		isSelected := i == selectedIndex
		prefix := "  "
		if isSelected {
			prefix = "> "
		}

		leftText := prefix + item.Label
		leftW := runewidth.StringWidth(leftText)

		var valDisplay string
		var rightStyled string

		if isSelected && editingDir {
			valDisplay = "[ " + dirInput.View() + " ]"
			rightStyled = styleDim.Render("[ ") + dirInput.View() + styleDim.Render(" ]")
		} else {
			rawVal := FormatSettingVal(item.Key, cfg)
			valDisplay = "[ " + rawVal + " ]"
			if isSelected {
				rightStyled = styleRegular.Render(valDisplay)
			} else {
				rightStyled = styleDim.Render(valDisplay)
			}
		}

		rightW := lipgloss.Width(valDisplay)
		gap := rowWidth - leftW - rightW
		if gap < 1 {
			gap = 1
		}

		var leftStyled string
		if isSelected {
			leftStyled = styleDim.Render("> ") + styleRegular.Render(item.Label)
		} else {
			leftStyled = "  " + styleRegular.Render(item.Label)
		}

		row := leftStyled + strings.Repeat(" ", gap) + rightStyled
		lines = append(lines, row)
	}

	return RenderBunCard("settings", width, strings.Join(lines, "\n"))
}
