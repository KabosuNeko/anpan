package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
	"github.com/KabosuNeko/anpan/internal/units"
)

type DestOption struct {
	Key      string
	Label    string
	Path     string
	IsCustom bool
}

func BuildDestOptions(defaultOutDir string) []DestOption {
	home, _ := os.UserHomeDir()
	cwd, _ := os.Getwd()

	cleanDef, _ := filepath.Abs(units.ResolveUserPath(defaultOutDir))
	cleanDownloads, _ := filepath.Abs(filepath.Join(home, "Downloads"))
	cleanVideos, _ := filepath.Abs(filepath.Join(home, "Videos"))
	cleanCwd, _ := filepath.Abs(cwd)

	var opts []DestOption
	opts = append(opts, DestOption{
		Key:   "default",
		Label: fmt.Sprintf("[↵] %s (default)", units.ShortenPath(defaultOutDir, home, 28)),
		Path:  defaultOutDir,
	})

	if cleanDownloads != cleanDef {
		opts = append(opts, DestOption{
			Key:   "d",
			Label: fmt.Sprintf("[D] %s", units.ShortenPath(filepath.Join(home, "Downloads"), home, 28)),
			Path:  filepath.Join(home, "Downloads"),
		})
	}
	if cleanVideos != cleanDef {
		opts = append(opts, DestOption{
			Key:   "v",
			Label: fmt.Sprintf("[V] %s", units.ShortenPath(filepath.Join(home, "Videos"), home, 28)),
			Path:  filepath.Join(home, "Videos"),
		})
	}
	if cleanCwd != cleanDef {
		opts = append(opts, DestOption{
			Key:   "c",
			Label: "[C] Current folder (./)",
			Path:  cwd,
		})
	}
	opts = append(opts, DestOption{
		Key:      "o",
		Label:    "[O] Custom path…",
		IsCustom: true,
	})

	return opts
}

func RenderDestView(width int, title string, subtitle string, isCustom bool, destIndex int, customInput textinput.Model, defaultOutDir string) string {
	var topBlock string
	if title != "" {
		h := lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(styleRegular.Render(units.Truncate(title, width)))
		s := lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(styleDim.Render(subtitle))
		topBlock = fmt.Sprintf("%s\n%s\n\n", h, s)
	}

	var content string
	if isCustom {
		prompt := styleDim.Render("enter directory path (↵ confirm, esc back):")
		inputStr := styleDim.Render("> ") + customInput.View()
		content = fmt.Sprintf("%s\n%s", prompt, inputStr)
	} else {
		opts := BuildDestOptions(defaultOutDir)
		var lines []string
		for i, o := range opts {
			if i == destIndex {
				lines = append(lines, styleDim.Render("❯ ")+styleRegular.Render(o.Label))
			} else {
				lines = append(lines, "  "+styleDim.Render(o.Label))
			}
		}
		content = strings.Join(lines, "\n")
	}

	card := RenderBunCard("choose save folder", width, content)
	return topBlock + card
}
