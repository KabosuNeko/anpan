package system

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type AnpanConfig struct {
	Aria2c         bool   `json:"aria2c"`
	Connections    int    `json:"connections"`
	AskSaveDir     bool   `json:"askSaveDir"`
	OutDir         string `json:"outDir"`
	PreferQuality  string `json:"preferQuality"`  // "ask" | "best" | "1080p" | "audio"
	VideoContainer string `json:"videoContainer"` // "mp4" | "mkv" | "webm"
	VideoCodec     string `json:"videoCodec"`     // "auto" | "av1" | "vp9" | "avc"
	AudioFormat    string `json:"audioFormat"`    // "mp3" | "m4a" | "opus" | "flac" | "wav"
	CookiesBrowser string `json:"cookiesBrowser"` // "none" | "chrome" | "firefox" | "brave" | "edge" | "safari"
	Subtitles      string `json:"subtitles"`      // "off" | "embed" | "write"
	SubLangs       string `json:"subLangs"`       // "vi,en" | "all" | "en"
	SponsorBlock   string `json:"sponsorBlock"`   // "off" | "remove" | "mark"
	EmbedMetadata  bool   `json:"embedMetadata"`
	WriteThumbnail bool   `json:"writeThumbnail"`
	Notifications  bool   `json:"notifications"`
	SpeedLimit     string `json:"speedLimit"` // "unlimited" | "1M" | "5M" | "10M" | "20M" | "50M"
	Lyrics         string `json:"lyrics"`     // "synced" | "off"
}

func DefaultConfig() AnpanConfig {
	home, _ := os.UserHomeDir()
	return AnpanConfig{
		Aria2c:         true,
		Connections:    16,
		AskSaveDir:     true,
		OutDir:         filepath.Join(home, "Downloads"),
		PreferQuality:  "ask",
		VideoContainer: "mp4",
		VideoCodec:     "auto",
		AudioFormat:    "mp3",
		CookiesBrowser: "none",
		Subtitles:      "off",
		SubLangs:       "vi,en",
		SponsorBlock:   "off",
		EmbedMetadata:  true,
		WriteThumbnail: false,
		Notifications:  true,
		SpeedLimit:     "unlimited",
		Lyrics:         "synced",
	}
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "anpan", "config.json")
}

func LoadConfig() AnpanConfig {
	cfg := DefaultConfig()
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	loaded := cfg
	if err := json.Unmarshal(data, &loaded); err == nil {
		if loaded.Connections != 4 && loaded.Connections != 8 && loaded.Connections != 16 && loaded.Connections != 32 {
			loaded.Connections = cfg.Connections
		}
		if loaded.OutDir == "" {
			loaded.OutDir = cfg.OutDir
		}
		if loaded.PreferQuality == "" {
			loaded.PreferQuality = cfg.PreferQuality
		}
		if loaded.VideoContainer == "" {
			loaded.VideoContainer = cfg.VideoContainer
		}
		if loaded.VideoCodec == "" {
			loaded.VideoCodec = cfg.VideoCodec
		}
		if loaded.AudioFormat == "" {
			loaded.AudioFormat = cfg.AudioFormat
		}
		if loaded.CookiesBrowser == "" {
			loaded.CookiesBrowser = cfg.CookiesBrowser
		}
		if loaded.Subtitles == "" {
			loaded.Subtitles = cfg.Subtitles
		}
		if loaded.SubLangs == "" {
			loaded.SubLangs = cfg.SubLangs
		}
		if loaded.SponsorBlock == "" {
			loaded.SponsorBlock = cfg.SponsorBlock
		}
		return loaded
	}
	return cfg
}

func SaveConfig(cfg AnpanConfig) error {
	p := configPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(data, '\n'), 0o644)
}
