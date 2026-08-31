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
