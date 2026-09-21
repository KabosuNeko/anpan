package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
	"github.com/KabosuNeko/anpan/internal/system"
	"github.com/KabosuNeko/anpan/internal/units"
)

type SettingItem struct {
	Key   string
	Label string
}

type SettingCategory struct {
	Title      string
	ShortTitle string
	Items      []SettingItem
}

var SettingCategories = []SettingCategory{
	{
		Title:      "General",
		ShortTitle: "General",
		Items: []SettingItem{
			{Key: "askSaveDir", Label: "ask save location"},
			{Key: "outDir", Label: "default save dir"},
			{Key: "colorTheme", Label: "color theme"},
			{Key: "notifications", Label: "desktop notifications"},
			{Key: "autoPaste", Label: "auto-paste clipboard"},
			{Key: "speedLimit", Label: "speed limit"},
		},
	},
	{
		Title:      "Video (yt-dlp)",
		ShortTitle: "Video",
		Items: []SettingItem{
			{Key: "videoContainer", Label: "video format (container)"},
			{Key: "videoCodec", Label: "video codec preference"},
			{Key: "preferQuality", Label: "auto-select quality"},
			{Key: "subtitles", Label: "subtitles"},
			{Key: "subLangs", Label: "subtitle languages"},
			{Key: "sponsorBlock", Label: "sponsorblock"},
			{Key: "cookiesBrowser", Label: "browser cookies"},
		},
	},
	{
		Title:      "Audio & Music",
		ShortTitle: "Audio",
		Items: []SettingItem{
			{Key: "audioFormat", Label: "audio format"},
			{Key: "embedMetadata", Label: "embed audio tags & cover"},
			{Key: "lyrics", Label: "download synced lyrics (.lrc)"},
			{Key: "writeThumbnail", Label: "write thumbnail image"},
		},
	},
	{
		Title:      "Torrent & aria2c",
		ShortTitle: "Torrent",
		Items: []SettingItem{
			{Key: "aria2c", Label: "aria2c accelerator"},
			{Key: "connections", Label: "aria2c connections (-x -s)"},
			{Key: "torrentSeedRatio", Label: "torrent seed ratio"},
			{Key: "defaultSearchCat", Label: "default search category"},
			{Key: "defaultSearchSort", Label: "default search sort"},
		},
	},
}

var (
	connectionChoices        = []int{4, 8, 16, 32}
	qualityChoices           = []string{"ask", "best", "1080p", "audio"}
	containerChoices         = []string{"mp4", "mkv", "webm"}
	codecChoices             = []string{"auto", "av1", "vp9", "avc"}
	audioChoices             = []string{"mp3", "m4a", "opus", "flac", "wav"}
	lyricsChoices            = []string{"synced", "off"}
	speedLimitChoices        = []string{"unlimited", "1M", "5M", "10M", "20M", "50M"}
	subtitleChoices          = []string{"off", "embed", "write"}
	sublangChoices           = []string{"vi,en", "all", "en"}
	sponsorChoices           = []string{"off", "remove", "mark"}
	cookiesChoices           = []string{"none", "chrome", "firefox", "brave", "edge", "safari"}
	torrentSeedRatioChoices  = []string{"off", "1.0", "2.0", "unlimited"}
	colorThemeChoices        = []string{"bakery", "terminal"}
	defaultSearchSortChoices = []string{"seeds", "size", "size-asc", "peers", "name", "source"}
	defaultSearchCatChoices  = []string{"all", "anime", "movies", "tv", "games"}
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

func onOff(v bool) string {
	if v {
		return "on"
	}
	return "off"
}

func FormatSettingVal(key string, cfg system.AnpanConfig) string {
	home, _ := os.UserHomeDir()
	switch key {
	case "aria2c":
		return onOff(cfg.Aria2c)
	case "askSaveDir":
		if cfg.AskSaveDir {
			return "always ask"
		}
		return "use default"
	case "embedMetadata":
		return onOff(cfg.EmbedMetadata)
	case "writeThumbnail":
		return onOff(cfg.WriteThumbnail)
	case "notifications":
		return onOff(cfg.Notifications)
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
	case "torrentSeedRatio":
		if cfg.TorrentSeedRatio == "" {
			return "off"
		}
		return cfg.TorrentSeedRatio
	case "colorTheme":
		if cfg.ColorTheme == "" {
			return "bakery"
		}
		return cfg.ColorTheme
	case "defaultSearchSort":
		if cfg.DefaultSearchSort == "" {
			return "seeds"
		}
		return cfg.DefaultSearchSort
	case "defaultSearchCat":
		if cfg.DefaultSearchCat == "" {
			return "all"
		}
		return cfg.DefaultSearchCat
	case "autoPaste":
		return onOff(cfg.AutoPaste)
	default:
		return ""
	}
}

// cycleChoice returns the entry dir steps from cur in choices, wrapping at both ends.
// When cur is not present it falls back to choices[def].
func cycleChoice[T comparable](choices []T, cur T, def int, dir int) T {
	idx := slices.Index(choices, cur)
	if idx < 0 {
		idx = def
	}
	return choices[(idx+dir+len(choices))%len(choices)]
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
		cfg.Connections = cycleChoice(connectionChoices, cfg.Connections, 2, dir)
	case "preferQuality":
		cfg.PreferQuality = cycleChoice(qualityChoices, cfg.PreferQuality, 0, dir)
	case "videoContainer":
		cfg.VideoContainer = cycleChoice(containerChoices, cfg.VideoContainer, 0, dir)
	case "videoCodec":
		cfg.VideoCodec = cycleChoice(codecChoices, cfg.VideoCodec, 0, dir)
	case "audioFormat":
		cfg.AudioFormat = cycleChoice(audioChoices, cfg.AudioFormat, 0, dir)
	case "subtitles":
		cfg.Subtitles = cycleChoice(subtitleChoices, cfg.Subtitles, 0, dir)
	case "subLangs":
		cfg.SubLangs = cycleChoice(sublangChoices, cfg.SubLangs, 0, dir)
	case "sponsorBlock":
		cfg.SponsorBlock = cycleChoice(sponsorChoices, cfg.SponsorBlock, 0, dir)
	case "cookiesBrowser":
		cfg.CookiesBrowser = cycleChoice(cookiesChoices, cfg.CookiesBrowser, 0, dir)
	case "speedLimit":
		cfg.SpeedLimit = cycleChoice(speedLimitChoices, cfg.SpeedLimit, 0, dir)
	case "lyrics":
		cfg.Lyrics = cycleChoice(lyricsChoices, cfg.Lyrics, 0, dir)
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
		cfg.OutDir = presets[(idx+dir+len(presets))%len(presets)]
	case "torrentSeedRatio":
		cfg.TorrentSeedRatio = cycleChoice(torrentSeedRatioChoices, cfg.TorrentSeedRatio, 0, dir)
	case "colorTheme":
		cfg.ColorTheme = cycleChoice(colorThemeChoices, cfg.ColorTheme, 0, dir)
		ApplyTheme(cfg.ColorTheme)
	case "defaultSearchSort":
		cfg.DefaultSearchSort = cycleChoice(defaultSearchSortChoices, cfg.DefaultSearchSort, 0, dir)
	case "defaultSearchCat":
		cfg.DefaultSearchCat = cycleChoice(defaultSearchCatChoices, cfg.DefaultSearchCat, 0, dir)
	case "autoPaste":
		cfg.AutoPaste = !cfg.AutoPaste
	}
}

func RenderSettingsView(width int, activeTab int, selectedIndex int, editingDir bool, dirInput textinput.Model, cfg system.AnpanConfig) string {
	rowWidth := max(width-4, 30)

	if activeTab < 0 || activeTab >= len(SettingCategories) {
		activeTab = 0
	}
	currentCat := SettingCategories[activeTab]

	var tabPills []string
	for i, cat := range SettingCategories {
		tabTitle := cat.ShortTitle
		if rowWidth >= 62 {
			tabTitle = cat.Title
		}
		if i == activeTab {
			tabPills = append(tabPills, styleTitle.Render("["+tabTitle+"]"))
		} else {
			tabPills = append(tabPills, styleDim.Render(" "+tabTitle+" "))
		}
	}
	tabLine := strings.Join(tabPills, " ")

	var lines []string
	lines = append(lines, tabLine)
	lines = append(lines, styleSubtle.Render(strings.Repeat("─", rowWidth)))

	for i, item := range currentCat.Items {
		isSelected := i == selectedIndex
		leftW := 2 + lipgloss.Width(item.Label)

		var rightStyled string
		var rightW int

		if isSelected && editingDir {
			rightW = lipgloss.Width("[ " + dirInput.View() + " ]")
			rightStyled = styleDim.Render("[ ") + dirInput.View() + styleDim.Render(" ]")
		} else {
			valDisplay := "[ " + FormatSettingVal(item.Key, cfg) + " ]"
			rightW = lipgloss.Width(valDisplay)
			if isSelected {
				rightStyled = styleRegular.Render(valDisplay)
			} else {
				rightStyled = styleDim.Render(valDisplay)
			}
		}

		gap := max(rowWidth-leftW-rightW, 1)

		var leftStyled string
		if isSelected {
			leftStyled = styleTitle.Render("> ") + styleRegular.Render(item.Label)
		} else {
			leftStyled = "  " + styleRegular.Render(item.Label)
		}

		row := leftStyled + strings.Repeat(" ", gap) + rightStyled
		lines = append(lines, row)
	}

	return RenderBunCard("settings", width, strings.Join(lines, "\n"))
}
