package tui

import (
	"testing"

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
}
