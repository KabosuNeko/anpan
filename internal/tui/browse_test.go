package tui

import (
	"strings"
	"testing"

	"github.com/KabosuNeko/anpan/internal/engine"
)

func TestGetSourceBadge(t *testing.T) {
	tests := []struct {
		src     string
		wantTag string
	}{
		{"yts", "YTS"},
		{"eztv", "EZTV"},
		{"nyaa", "NYAA"},
		{"subsplease", "SUB"},
		{"tpb", "TPB"},
		{"thepiratebay", "TPB"},
		{"1337x", "1337"},
		{"fitgirl", "FITG"},
		{"fitg", "FITG"},
	}

	for _, tc := range tests {
		tag, _ := GetSourceBadge(tc.src)
		if tag != tc.wantTag {
			t.Errorf("GetSourceBadge(%q) = %q, want %q", tc.src, tag, tc.wantTag)
		}
	}
}

func TestFormatSortLabel(t *testing.T) {
	tests := []struct {
		mode string
		want string
	}{
		{"seeds", "seeds ↓"},
		{"size", "size ↓"},
		{"size-asc", "size ↑"},
		{"peers", "peers ↓"},
		{"name", "name A–Z"},
		{"source", "source A–Z"},
	}

	for _, tc := range tests {
		got := FormatSortLabel(tc.mode)
		if got != tc.want {
			t.Errorf("FormatSortLabel(%q) = %q, want %q", tc.mode, got, tc.want)
		}
	}
}

func TestRenderBrowseViewModes(t *testing.T) {
	results := []engine.TorrentSearchResult{
		{
			Title:     "Oppenheimer (2023) [1080p]",
			SizeBytes: 2100000000,
			Seeders:   1240,
			Leechers:  88,
			Source:    "yts",
			Magnet:    "magnet:?xt=urn:btih:1111111111111111111111111111111111111111",
		},
		{
			Title:     "Frieren: Beyond Journey's End - 28",
			SizeBytes: 1400000000,
			Seeders:   500,
			Leechers:  20,
			Source:    "subsplease",
			Magnet:    "magnet:?xt=urn:btih:2222222222222222222222222222222222222222",
		},
	}

	// 1. Two-column view (width 80)
	out80 := RenderBrowseView(BrowseViewState{
		Width:          80,
		Height:         24,
		ActiveCategory: "all",
		FocusedRegion:  "content",
		Results:        results,
		Cursor:         0,
		SortMode:       "seeds",
	})
	if !strings.Contains(out80, "Oppenheimer") {
		t.Errorf("Expected Oppenheimer in two-column output")
	}
	if !strings.Contains(out80, "All") || !strings.Contains(out80, "Anime") || !strings.Contains(out80, "Games") {
		t.Errorf("Expected sidebar categories (including Games) in two-column output")
	}

	// 2. Narrow stacked view (width 60)
	out60 := RenderBrowseView(BrowseViewState{
		Width:          60,
		Height:         24,
		ActiveCategory: "anime",
		FocusedRegion:  "search",
		SearchInput:    "frieren",
		Results:        results,
		Cursor:         1,
	})
	if !strings.Contains(out60, "frieren") {
		t.Errorf("Expected search input in narrow output")
	}

	// 3. Detail view
	outDetail := RenderBrowseView(BrowseViewState{
		Width:         80,
		Height:        24,
		FocusedRegion: "detail",
		DetailResult:  &results[0],
	})
	if !strings.Contains(outDetail, "1240 seeders") {
		t.Errorf("Expected health stats in detail output")
	}
	if !strings.Contains(outDetail, "Folder") || !strings.Contains(outDetail, "Close") {
		t.Errorf("Expected Folder and Close in detail action bar")
	}

	// 4. Seeding view
	outSeeding := RenderBrowseView(BrowseViewState{
		Width:          80,
		Height:         24,
		ActiveCategory: "seeding",
		FocusedRegion:  "content",
		SeedingActive:  true,
		SeedingName:    "my_cool_archive.zip",
	})
	if !strings.Contains(outSeeding, "my_cool_archive.zip") {
		t.Errorf("Expected seeding name in seeding output")
	}

	// 5. Min seeds filter view
	outFiltered := RenderBrowseView(BrowseViewState{
		Width:          80,
		Height:         24,
		ActiveCategory: "all",
		FocusedRegion:  "content",
		Results:        results,
		MinSeeds:       5,
	})
	if !strings.Contains(outFiltered, "min seeds: 5+") {
		t.Errorf("Expected 'min seeds: 5+' in filtered output")
	}
	if !strings.Contains(outFiltered, "[≥5s]") {
		t.Errorf("Expected '[≥5s]' in filtered panel title")
	}
}
