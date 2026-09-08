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

	if !testing.Short() {
		resKemono, err := InspectTarget(ctx, "https://kemono.cr/patreon/user/49965584/post/142699525")
		if err != nil || resKemono.Type != TargetArchive || resKemono.ArchivePost == nil {
			t.Errorf("InspectTarget kemono failed: %+v, err: %v", resKemono, err)
		}

		resCoomer, err := InspectTarget(ctx, "https://coomer.st/onlyfans/user/anaimiya/post/1942401985")
		if err != nil || resCoomer.Type != TargetArchive || resCoomer.ArchivePost == nil {
			t.Errorf("InspectTarget coomer failed: %+v, err: %v", resCoomer, err)
		}
	}
}

func TestInspectTargetTorlinkFeatures(t *testing.T) {
	ctx := context.Background()

	// 1. Bare infohash
	ih := "4a3f5e08bcef825718eda30637230585e3330599"
	resIH, err := InspectTarget(ctx, ih)
	if err != nil || resIH.Type != TargetTorrent {
		t.Fatalf("expected TargetTorrent for bare infohash: %+v, err: %v", resIH, err)
	}

	// 2. Search query with '?'
	resSearch, err := InspectTarget(ctx, "?frieren beyond")
	if err != nil || resSearch.Type != TargetSearch || resSearch.SearchQuery != "frieren beyond" {
		t.Fatalf("expected TargetSearch for ? query: %+v, err: %v", resSearch, err)
	}

	// 3. Multi-word search query
	resMulti, err := InspectTarget(ctx, "one piece 1100")
	if err != nil || resMulti.Type != TargetSearch || resMulti.SearchQuery != "one piece 1100" {
		t.Fatalf("expected TargetSearch for multi-word query: %+v, err: %v", resMulti, err)
	}

	// 4. Dragged file path with quotes and spaces
	clean := CleanLocalPath("'/home/user/My Videos/sample.mp4'")
	if clean != "/home/user/My Videos/sample.mp4" {
		t.Errorf("CleanLocalPath failed: got %q", clean)
	}
}
