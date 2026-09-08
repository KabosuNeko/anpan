package tui

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
)

func TestRenderTrayInputWidth(t *testing.T) {
	totalWidth := 64
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "https://... or magnet:?..."
	ti.Focus()
	ti.SetWidth(totalWidth - 12)

	tray := RenderTrayInput("url / magnet / file", totalWidth, ti.View(), "bake", true)
	lines := strings.Split(tray, "\n")
	for i, l := range lines {
		w := lipgloss.Width(l)
		if w != totalWidth {
			t.Errorf("line %d has width %d, expected %d: %q", i, w, totalWidth, l)
		}
	}

	// Test with long URL that exceeds width
	ti.SetValue("https://www.youtube.com/watch?v=tlFnfEWZCtQ&list=RDtlFnfEWZCtQ&start_radio=1&extra_very_long_parameter_that_overflows_completely")
	trayLong := RenderTrayInput("url / magnet / file", totalWidth, ti.View(), "bake", false)
	linesLong := strings.Split(trayLong, "\n")
	for i, l := range linesLong {
		w := lipgloss.Width(l)
		if w != totalWidth {
			t.Errorf("long link line %d has width %d, expected %d: %q", i, w, totalWidth, l)
		}
	}
}

func TestRenderMascotWidth(t *testing.T) {
	totalWidth := 64
	mascot := RenderMascot(totalWidth)
	lines := strings.Split(mascot, "\n")
	for i, l := range lines {
		w := lipgloss.Width(l)
		if w != totalWidth {
			t.Errorf("line %d has width %d, expected %d", i, w, totalWidth)
		}
	}
}

func TestRenderFooterHintsSingleLine(t *testing.T) {
	hints := stageHints[StageInput]
	rendered := RenderFooterHints(hints)
	w := lipgloss.Width(rendered)
	if w > 70 {
		t.Errorf("RenderFooterHints for StageInput width %d exceeds 70, which causes wrapping", w)
	}

	footer := lipgloss.NewStyle().Width(70).Align(lipgloss.Center).Render(rendered)
	lines := strings.Split(footer, "\n")
	if len(lines) != 1 {
		t.Errorf("Expected footer to be a single line, got %d lines: %v", len(lines), lines)
	}

	// Test responsive tier 1 (< 70 width)
	tier1Hints := [][2]string{
		{"↵", "bake"},
		{"^f", "search torrent"},
		{"^e", "seed"},
		{"^s", "settings"},
		{"^c", "quit"},
	}
	rTier1 := RenderFooterHints(tier1Hints)
	if lipgloss.Width(rTier1) > 58 {
		t.Errorf("Tier 1 hints width %d exceeds 58", lipgloss.Width(rTier1))
	}

	// Test responsive tier 2 (< 58 width)
	tier2Hints := [][2]string{
		{"↵", "bake"},
		{"^f", "search torrent"},
		{"^c", "quit"},
	}
	rTier2 := RenderFooterHints(tier2Hints)
	if lipgloss.Width(rTier2) > 36 {
		t.Errorf("Tier 2 hints width %d exceeds 36", lipgloss.Width(rTier2))
	}
}

