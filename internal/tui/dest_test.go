package tui

import (
	"testing"
)

func TestBuildDestOptions(t *testing.T) {
	opts := BuildDestOptions("~/Downloads")
	if len(opts) < 2 {
		t.Fatalf("expected at least 2 options, got %d", len(opts))
	}
	if opts[0].Key != "default" {
		t.Errorf("expected first option to be default, got %s", opts[0].Key)
	}

	last := opts[len(opts)-1]
	if !last.IsCustom || last.Key != "o" {
		t.Errorf("expected last option to be custom 'o', got %+v", last)
	}
}
