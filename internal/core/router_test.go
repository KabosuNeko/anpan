package core

import (
	"context"
	"regexp"
	"testing"
)

func TestParseMagnetName(t *testing.T) {
	m1 := "magnet:?xt=urn:btih:d540fc48eb12f2833163eed6421d449dd8f1ce1f&dn=NixOS+24.11+Minimal"
	if got := ParseMagnetName(m1); got != "NixOS 24.11 Minimal" {
		t.Errorf("ParseMagnetName(m1) = %q, want 'NixOS 24.11 Minimal'", got)
	}

	m2 := "magnet:?xt=urn:btih:1234567890abcdef"
	got2 := ParseMagnetName(m2)
	re := regexp.MustCompile(`(?i)Torrent \([0-9a-f]+\)`)
	if !re.MatchString(got2) {
		t.Errorf("ParseMagnetName(m2) = %q, want Torrent (hash)", got2)
	}
}

func TestInspectTarget(t *testing.T) {
	ctx := context.Background()

	res1, err := InspectTarget(ctx, "magnet:?xt=urn:btih:1234567890abcdef&dn=NixOS")
	if err != nil || res1.Type != TargetTorrent || res1.Name != "NixOS" {
		t.Errorf("InspectTarget magnet failed: %+v, err: %v", res1, err)
	}

	res2, err := InspectTarget(ctx, "https://releases.nixos.org/nixos/24.11/nixos-24.11.torrent")
	if err != nil || res2.Type != TargetTorrent || res2.Name != "nixos-24.11" {
		t.Errorf("InspectTarget torrent failed: %+v, err: %v", res2, err)
	}

	res3, err := InspectTarget(ctx, "https://music.youtube.com/watch?v=123")
	if err != nil || res3.Type != TargetVideo || res3.CleanURL != "https://music.youtube.com/watch?v=123" {
		t.Errorf("InspectTarget video failed: %+v, err: %v", res3, err)
	}

	res4, err := InspectTarget(ctx, "https://channels.nixos.org/nixos-24.11/latest-nixos-minimal-x86_64-linux.iso")
	if err != nil || res4.Type != TargetDirect || res4.Filename != "latest-nixos-minimal-x86_64-linux.iso" {
		t.Errorf("InspectTarget direct failed: %+v, err: %v", res4, err)
	}
}
