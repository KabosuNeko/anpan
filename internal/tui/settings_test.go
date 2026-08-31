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
}

func TestFormatSettingVal(t *testing.T) {
	cfg := system.DefaultConfig()
	if val := FormatSettingVal("aria2c", cfg); val != "on" {
		t.Errorf("expected 'on', got %s", val)
	}
	if val := FormatSettingVal("connections", cfg); val != "16" {
		t.Errorf("expected '16', got %s", val)
	}
}
