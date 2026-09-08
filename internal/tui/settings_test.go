package tui

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/textinput"
	"github.com/KabosuNeko/anpan/internal/system"
)

func TestCycleConfig(t *testing.T) {
	cfg := system.DefaultConfig()

	// Toggle Aria2c
	CycleConfig(&cfg, "aria2c", 1)
	if cfg.Aria2c {
		t.Errorf("expected Aria2c to be toggled to false")
	}

	// Connections cycle
	cfg.Connections = 16
	CycleConfig(&cfg, "connections", 1)
	if cfg.Connections != 32 {
		t.Errorf("expected Connections to be 32, got %d", cfg.Connections)
	}

	// Quality cycle
	cfg.PreferQuality = "ask"
	CycleConfig(&cfg, "preferQuality", 1)
	if cfg.PreferQuality != "best" {
		t.Errorf("expected PreferQuality to be best, got %s", cfg.PreferQuality)
	}
	// Video codec cycle
	cfg.VideoCodec = "auto"
	CycleConfig(&cfg, "videoCodec", 1)
	if cfg.VideoCodec != "av1" {
		t.Errorf("expected VideoCodec to be av1, got %s", cfg.VideoCodec)
	}

	// Notifications cycle
	cfg.Notifications = true
	CycleConfig(&cfg, "notifications", 1)
	if cfg.Notifications {
		t.Errorf("expected Notifications to be toggled to false")
	}

	// Speed limit cycle
	cfg.SpeedLimit = "unlimited"
	CycleConfig(&cfg, "speedLimit", 1)
	if cfg.SpeedLimit != "1M" {
		t.Errorf("expected SpeedLimit to be 1M, got %s", cfg.SpeedLimit)
	}

	// Lyrics cycle
	cfg.Lyrics = "synced"
	CycleConfig(&cfg, "lyrics", 1)
	if cfg.Lyrics != "off" {
		t.Errorf("expected Lyrics to be off, got %s", cfg.Lyrics)
	}

	// Torrent seed ratio cycle
	cfg.TorrentSeedRatio = "off"
	CycleConfig(&cfg, "torrentSeedRatio", 1)
	if cfg.TorrentSeedRatio != "1.0" {
		t.Errorf("expected TorrentSeedRatio to be 1.0, got %s", cfg.TorrentSeedRatio)
	}

	// Color theme cycle
	cfg.ColorTheme = "bakery"
	CycleConfig(&cfg, "colorTheme", 1)
	if cfg.ColorTheme != "terminal" {
		t.Errorf("expected ColorTheme to be terminal, got %s", cfg.ColorTheme)
	}
	CycleConfig(&cfg, "colorTheme", 1)
	if cfg.ColorTheme != "bakery" {
		t.Errorf("expected ColorTheme to be bakery, got %s", cfg.ColorTheme)
	}

	// Default search sort cycle
	cfg.DefaultSearchSort = "seeds"
	CycleConfig(&cfg, "defaultSearchSort", 1)
	if cfg.DefaultSearchSort != "size" {
		t.Errorf("expected DefaultSearchSort to be size, got %s", cfg.DefaultSearchSort)
	}

	// Default search category cycle
	cfg.DefaultSearchCat = "all"
	CycleConfig(&cfg, "defaultSearchCat", 1)
	if cfg.DefaultSearchCat != "anime" {
		t.Errorf("expected DefaultSearchCat to be anime, got %s", cfg.DefaultSearchCat)
	}

	// AutoPaste toggle
	cfg.AutoPaste = false
	CycleConfig(&cfg, "autoPaste", 1)
	if !cfg.AutoPaste {
		t.Errorf("expected AutoPaste to be true")
	}
}

func TestFormatSettingVal(t *testing.T) {
	cfg := system.DefaultConfig()
	if val := FormatSettingVal("aria2c", cfg); val != "on" {
		t.Errorf("expected 'on', got %s", val)
	}
	if val := FormatSettingVal("connections", cfg); val != "16" {
		t.Errorf("expected '16', got %s", val)
	}
	if val := FormatSettingVal("notifications", cfg); val != "on" {
		t.Errorf("expected 'on', got %s", val)
	}
	if val := FormatSettingVal("speedLimit", cfg); val != "unlimited" {
		t.Errorf("expected 'unlimited', got %s", val)
	}
	if val := FormatSettingVal("lyrics", cfg); val != "synced" {
		t.Errorf("expected 'synced', got %s", val)
	}
	if val := FormatSettingVal("videoCodec", cfg); val != "auto" {
		t.Errorf("expected 'auto', got %s", val)
	}
	if val := FormatSettingVal("torrentSeedRatio", cfg); val != "off" {
		t.Errorf("expected 'off', got %s", val)
	}
	if val := FormatSettingVal("colorTheme", cfg); val != "bakery" {
		t.Errorf("expected 'bakery', got %s", val)
	}
	if val := FormatSettingVal("defaultSearchSort", cfg); val != "seeds" {
		t.Errorf("expected 'seeds', got %s", val)
	}
	if val := FormatSettingVal("defaultSearchCat", cfg); val != "all" {
		t.Errorf("expected 'all', got %s", val)
	}
	if val := FormatSettingVal("autoPaste", cfg); val != "off" {
		t.Errorf("expected 'off', got %s", val)
	}
}

func TestSettingsCategories(t *testing.T) {
	if len(SettingCategories) != 4 {
		t.Fatalf("expected 4 setting categories, got %d", len(SettingCategories))
	}

	expectedKeys := map[string]bool{
		"general": true,
		"video":   true,
		"audio":   true,
		"torrent": true,
	}

	totalItems := 0
	for _, cat := range SettingCategories {
		if !expectedKeys[cat.Key] {
			t.Errorf("unexpected category key: %s", cat.Key)
		}
		if len(cat.Items) == 0 {
			t.Errorf("category %s has no items", cat.Key)
		}
		totalItems += len(cat.Items)
	}

	if totalItems != len(SettingItems) {
		t.Errorf("expected total category items %d to match SettingItems %d", totalItems, len(SettingItems))
	}
}

func TestRenderSettingsViewTabbed(t *testing.T) {
	cfg := system.DefaultConfig()

	// Tab 0: General
	outGen := RenderSettingsView(66, 0, 0, false, textinput.Model{}, cfg)
	if !strings.Contains(outGen, "General") {
		t.Errorf("expected 'General' tab in output")
	}
	if !strings.Contains(outGen, "ask save location") {
		t.Errorf("expected 'ask save location' in General tab")
	}
	if strings.Contains(outGen, "video format (container)") {
		t.Errorf("did not expect video settings in General tab")
	}

	// Tab 1: Video
	outVid := RenderSettingsView(66, 1, 0, false, textinput.Model{}, cfg)
	if !strings.Contains(outVid, "Video") {
		t.Errorf("expected 'Video' tab in output")
	}
	if !strings.Contains(outVid, "video format (container)") {
		t.Errorf("expected video container setting in Video tab")
	}

	// Tab 2: Audio
	outAud := RenderSettingsView(66, 2, 0, false, textinput.Model{}, cfg)
	if !strings.Contains(outAud, "Audio") {
		t.Errorf("expected 'Audio' tab in output")
	}
	if !strings.Contains(outAud, "audio format") {
		t.Errorf("expected audio format setting in Audio tab")
	}

	// Tab 3: Torrent
	outTor := RenderSettingsView(66, 3, 0, false, textinput.Model{}, cfg)
	if !strings.Contains(outTor, "Torrent") {
		t.Errorf("expected 'Torrent' tab in output")
	}
	if !strings.Contains(outTor, "torrent seed ratio") {
		t.Errorf("expected torrent seed ratio in Torrent tab")
	}
}
