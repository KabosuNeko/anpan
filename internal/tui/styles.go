package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"
)

var (
	colorBreadCrust = lipgloss.Color("#cb904d")
	colorBreadDough = lipgloss.Color("#f6d5a8")
	colorDim        = lipgloss.Color("#70685e")
	colorSubtle     = lipgloss.Color("#4a453e")
	colorWhite      = lipgloss.Color("#dcd8d0")
	colorSuccess    = lipgloss.Color("#73b06f")
	colorError      = lipgloss.Color("#d95d5d")

	styleTitle = lipgloss.NewStyle().
			Foreground(colorBreadCrust).
			Bold(true)

	styleRegular = lipgloss.NewStyle().
			Foreground(colorWhite)

	styleDim = lipgloss.NewStyle().
			Foreground(colorDim)

	styleSubtle = lipgloss.NewStyle().
			Foreground(colorSubtle)

	styleSelected = lipgloss.NewStyle().
			Foreground(colorBreadDough).
			Bold(true)

	styleSuccess = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Bold(true)

	styleError = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)
)

func ApplyTheme(theme string) {
	if strings.ToLower(strings.TrimSpace(theme)) == "terminal" {
		colorBreadCrust = lipgloss.Color("3")  // ANSI 3 Yellow
		colorBreadDough = lipgloss.Color("11") // ANSI 11 Bright Yellow
		colorDim = lipgloss.Color("8")         // ANSI 8 Gray
		colorSubtle = lipgloss.Color("8")
		colorWhite = lipgloss.Color("7")   // ANSI 7 Default Foreground
		colorSuccess = lipgloss.Color("2") // ANSI 2 Green
		colorError = lipgloss.Color("1")   // ANSI 1 Red

		styleTitle = lipgloss.NewStyle().Foreground(colorBreadCrust).Bold(true)
		styleRegular = lipgloss.NewStyle()
		styleDim = lipgloss.NewStyle().Faint(true)
		styleSubtle = lipgloss.NewStyle().Faint(true)
		styleSelected = lipgloss.NewStyle().Foreground(colorBreadDough).Bold(true)
		styleSuccess = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
		styleError = lipgloss.NewStyle().Foreground(colorError).Bold(true)
	} else {
		colorBreadCrust = lipgloss.Color("#cb904d")
		colorBreadDough = lipgloss.Color("#f6d5a8")
		colorDim = lipgloss.Color("#70685e")
		colorSubtle = lipgloss.Color("#4a453e")
		colorWhite = lipgloss.Color("#dcd8d0")
		colorSuccess = lipgloss.Color("#73b06f")
		colorError = lipgloss.Color("#d95d5d")

		styleTitle = lipgloss.NewStyle().Foreground(colorBreadCrust).Bold(true)
		styleRegular = lipgloss.NewStyle().Foreground(colorWhite)
		styleDim = lipgloss.NewStyle().Foreground(colorDim)
		styleSubtle = lipgloss.NewStyle().Foreground(colorSubtle)
		styleSelected = lipgloss.NewStyle().Foreground(colorBreadDough).Bold(true)
		styleSuccess = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
		styleError = lipgloss.NewStyle().Foreground(colorError).Bold(true)
	}
}

func RenderTrayInput(title string, totalWidth int, inputContent string, actionLabel string, actionDim bool) string {
	btnW := len(actionLabel) + 4
	leftW := totalWidth - btnW
	titleW := runewidth.StringWidth(title)
	leftInner := max(leftW-titleW-4, 1)

	topBorder := styleDim.Render("╭─ ") + styleRegular.Render(title) + styleDim.Render(" "+strings.Repeat("─", leftInner)+"┬"+strings.Repeat("─", btnW-2)+"╮")

	innerInputW := max(leftW-4, 1)

	// Truncate/clip input content to innerInputW so long text cannot break the box
	clippedContent := lipgloss.NewStyle().MaxWidth(innerInputW).Render(inputContent)
	padLen := max(innerInputW-lipgloss.Width(clippedContent), 0)

	leftLine := styleDim.Render("│ > ") + clippedContent + strings.Repeat(" ", padLen)

	btnStyle := styleTitle
	if actionDim {
		btnStyle = styleDim
	}
	midLine := leftLine + styleDim.Render("│ ") + btnStyle.Render(actionLabel) + styleDim.Render(" │")

	botBorder := styleDim.Render("╰" + strings.Repeat("─", leftW-1) + "┴" + strings.Repeat("─", btnW-2) + "╯")

	return fmt.Sprintf("%s\n%s\n%s", topBorder, midLine, botBorder)
}

func RenderBunCard(title string, totalWidth int, content string) string {
	return renderRoundedBox(title, max(totalWidth, 30), strings.Split(content, "\n"), false)
}

func RenderCrustBar(percent float64, width int) string {
	if width < 10 {
		width = 20
	}
	clamped := min(max(percent, 0), 1)

	filled := int(clamped * float64(width))
	empty := max(width-filled, 0)

	filledStr := styleTitle.Render(strings.Repeat("█", filled))
	emptyStr := styleSubtle.Render(strings.Repeat("░", empty))
	pctStr := styleTitle.Render(fmt.Sprintf(" %3.0f%%", clamped*100))

	return filledStr + emptyStr + pctStr
}

func RenderFooterHints(hints [][2]string) string {
	var parts []string
	for _, h := range hints {
		parts = append(parts, styleRegular.Render(h[0])+" "+styleDim.Render(h[1]))
	}
	return strings.Join(parts, "  ")
}
