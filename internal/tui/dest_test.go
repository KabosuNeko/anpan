package tui

import (
	"testing"
)

func TestBuildDestOptions(t *testing.T) {
	opts := BuildDestOptions("~/Downloads")
	if len(opts) < 2 {
		t.Fatalf("expected at least 2 options, got %d", len(opts))
	}
	if opts[0].Path != "~/Downloads" {
		t.Errorf("expected first option to be the default dir, got %q", opts[0].Path)
	}

	last := opts[len(opts)-1]
	if !last.IsCustom {
		t.Errorf("expected last option to be custom, got %+v", last)
	}
}
