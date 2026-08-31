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

func RenderTrayInput(title string, totalWidth int, inputContent string, actionLabel string, actionDim bool) string {
	btnW := len(actionLabel) + 4
	if actionLabel == "" {
		btnW = 0
	}
	leftW := totalWidth - btnW
	titleW := runewidth.StringWidth(title)
	leftInner := leftW - titleW - 4
	if leftInner < 1 {
		leftInner = 1
	}

	var topBorder string
	if btnW > 0 {
		topBorder = styleDim.Render("╭─ ") + styleRegular.Render(title) + styleDim.Render(" "+strings.Repeat("─", leftInner)+"┬"+strings.Repeat("─", btnW-2)+"╮")
	} else {
		topBorder = styleDim.Render("╭─ ") + styleRegular.Render(title) + styleDim.Render(" "+strings.Repeat("─", leftInner)+"╮")
	}

	// Inner input width available
	innerInputW := leftW - 4
	if innerInputW < 1 {
		innerInputW = 1
	}

	// Truncate/clip input content to innerInputW so long text cannot break the box
	clippedContent := lipgloss.NewStyle().MaxWidth(innerInputW).Render(inputContent)
	contentW := lipgloss.Width(clippedContent)
	padLen := innerInputW - contentW
	if padLen < 0 {
		padLen = 0
	}

	leftLine := styleDim.Render("│ > ") + clippedContent + strings.Repeat(" ", padLen)

	var midLine string
	if btnW > 0 {
		btnStyle := styleTitle
		if actionDim {
			btnStyle = styleDim
		}
		midLine = leftLine + styleDim.Render("│ ") + btnStyle.Render(actionLabel) + styleDim.Render(" │")
	} else {
		midLine = leftLine + styleDim.Render(" │")
	}

	var botBorder string
	if btnW > 0 {
		botBorder = styleDim.Render("╰" + strings.Repeat("─", leftW-1) + "┴" + strings.Repeat("─", btnW-2) + "╯")
	} else {
		botBorder = styleDim.Render("╰" + strings.Repeat("─", leftW-1) + "╯")
	}

	return fmt.Sprintf("%s\n%s\n%s", topBorder, midLine, botBorder)
}

func RenderBunCard(title string, totalWidth int, content string) string {
	if totalWidth < 30 {
		totalWidth = 30
	}
	inner := totalWidth - 2
	tail := inner - runewidth.StringWidth(title) - 3
	if tail < 0 {
		tail = 0
	}

	topBorder := styleDim.Render("╭─ ") + styleRegular.Render(title) + styleDim.Render(" "+strings.Repeat("─", tail)+"╮")

	lines := strings.Split(content, "\n")
	var body []string
	contentMaxW := inner - 2
	if contentMaxW < 1 {
		contentMaxW = 1
	}
	for _, l := range lines {
		clipped := lipgloss.NewStyle().MaxWidth(contentMaxW).Render(l)
		w := lipgloss.Width(clipped)
		pad := contentMaxW - w
		if pad < 0 {
			pad = 0
		}
		body = append(body, styleDim.Render("│ ")+clipped+strings.Repeat(" ", pad)+styleDim.Render(" │"))
	}
	bottomBorder := styleDim.Render("╰" + strings.Repeat("─", inner) + "╯")

	return fmt.Sprintf("%s\n%s\n%s", topBorder, strings.Join(body, "\n"), bottomBorder)
}

func RenderCrustBar(percent float64, width int) string {
	if width < 10 {
		width = 20
	}
	clamped := percent
	if clamped < 0 {
		clamped = 0
	} else if clamped > 1 {
		clamped = 1
	}

	filled := int(clamped * float64(width))
	empty := width - filled
	if empty < 0 {
		empty = 0
	}

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
	return strings.Join(parts, "   ")
}
