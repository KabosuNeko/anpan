package engine

import (
	"strings"
	"testing"
)

func TestIsInfoHash(t *testing.T) {
	// 40-char hex
	hex40 := "4a3f5e08bcef825718eda30637230585e3330599"
	if !IsInfoHash(hex40) {
		t.Errorf("expected %s to be recognized as InfoHash", hex40)
	}

	// 32-char Base32
	b32 := "JJ7V4CG456BFOGHNUYDDOMYFQXPTHBNZ"
	if !IsInfoHash(b32) {
		t.Errorf("expected %s to be recognized as Base32 InfoHash", b32)
	}

	// Invalid
	invalid := "notaninfohash"
	if IsInfoHash(invalid) {
		t.Errorf("expected %s not to be recognized as InfoHash", invalid)
	}
}

func TestNormalizeInfoHash(t *testing.T) {
	hex40 := "4A3F5E08BCEF825718EDA30637230585E3330599"
	norm, err := NormalizeInfoHash(hex40)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if norm != strings.ToLower(hex40) {
		t.Errorf("expected lowercase hex, got %s", norm)
	}

	b32 := "JJ7V4CG456BFOGHNUYDDOMYFQXPTHBNZ"
	normB32, err := NormalizeInfoHash(b32)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(normB32) != 40 {
		t.Errorf("expected 40-char hex from Base32, got %s", normB32)
	}
}

func TestBuildMagnet(t *testing.T) {
	ih := "4a3f5e08bcef825718eda30637230585e3330599"
	mag := BuildMagnet(ih, "Test Video", nil)
	if !strings.HasPrefix(mag, "magnet:?xt=urn:btih:"+ih) {
		t.Errorf("expected magnet to start with btih, got %s", mag)
	}
	if !strings.Contains(mag, "dn=Test+Video") {
		t.Errorf("expected dn param in magnet, got %s", mag)
	}
	if !strings.Contains(mag, "tr=udp%3A%2F%2Ftracker.opentrackr.org") {
		t.Errorf("expected tracker in magnet, got %s", mag)
	}
}

func TestEnrichMagnetWithTrackers(t *testing.T) {
	bareMag := "magnet:?xt=urn:btih:4a3f5e08bcef825718eda30637230585e3330599&dn=Ubuntu"
	enriched := EnrichMagnetWithTrackers(bareMag)
	if !strings.Contains(enriched, "tr=") {
		t.Errorf("expected enriched magnet to contain trackers, got %s", enriched)
	}
}
