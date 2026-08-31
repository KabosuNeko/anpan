package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

var mascotLines = []string{
	"     .,cdxkkxoc,.             ",
	"  'cdkdxxdkkdxkkkd:.          ",
	".dkkkkdxxdkdxkxxkkkko         ",
	"okkkkkkkkkkkkkkOKXXXXXK0Okc.  ",
	" dkkkkkkkkkkx0WNNK000000KXNWk ",
	"   .kkkkkkkkOMKdc::::::::cd0Mx",
	"             'xc:::::::::::d; ",
}

func RenderMascot(width int) string {
	var centered []string
	centerStyle := lipgloss.NewStyle().Width(width).Align(lipgloss.Center)
	for i, line := range mascotLines {
		var styledLine string
		if i == 0 {
			styledLine = styleDim.Render(line)
		} else if i < 4 {
			styledLine = styleTitle.Render(line)
		} else {
			styledLine = styleSelected.Render(line)
		}
		centered = append(centered, centerStyle.Render(styledLine))
	}
	return strings.Join(centered, "\n")
}
