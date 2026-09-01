package system

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Aria2c {
		t.Errorf("expected Aria2c true by default")
	}
	if cfg.Connections != 16 {
		t.Errorf("expected Connections 16 by default, got %d", cfg.Connections)
	}
	if cfg.VideoContainer != "mp4" {
		t.Errorf("expected mp4 container by default, got %s", cfg.VideoContainer)
	}
	if cfg.VideoCodec != "auto" {
		t.Errorf("expected auto videoCodec by default, got %s", cfg.VideoCodec)
	}
	if cfg.AudioFormat != "mp3" {
		t.Errorf("expected mp3 audioFormat by default, got %s", cfg.AudioFormat)
	}
	if !cfg.Notifications {
		t.Errorf("expected Notifications true by default")
	}
	if cfg.SpeedLimit != "unlimited" {
		t.Errorf("expected SpeedLimit unlimited by default, got %s", cfg.SpeedLimit)
	}
	if cfg.Lyrics != "synced" {
		t.Errorf("expected Lyrics synced by default, got %s", cfg.Lyrics)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	isolateTestHome(t)

	cfg := DefaultConfig()
	cfg.Connections = 32
	cfg.PreferQuality = "1080p"
	cfg.AudioFormat = "flac"

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	loaded := LoadConfig()
	if loaded.Connections != 32 {
		t.Errorf("expected 32 connections, got %d", loaded.Connections)
	}
	if loaded.PreferQuality != "1080p" {
		t.Errorf("expected preferQuality 1080p, got %s", loaded.PreferQuality)
	}
	if loaded.AudioFormat != "flac" {
		t.Errorf("expected audioFormat flac, got %s", loaded.AudioFormat)
	}
}
