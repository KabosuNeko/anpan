package tui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/KabosuNeko/anpan/internal/engine"
	"github.com/KabosuNeko/anpan/internal/units"
)

type BrowseCategory struct {
	Key   string
	Label string
}

var BrowseCategories = []BrowseCategory{
	{Key: "all", Label: "All"},
	{Key: "anime", Label: "Anime"},
	{Key: "movies", Label: "Movies"},
	{Key: "tv", Label: "TV"},
	{Key: "games", Label: "Games"},
	{Key: "seeding", Label: "Seeding"},
}

func GetSourceBadge(src string) (string, color.Color) {
	tag := strings.ToUpper(src)
	switch strings.ToLower(src) {
	case "yts":
		return "YTS", colorBreadDough
	case "eztv":
		return "EZTV", colorBreadCrust
	case "nyaa":
		return "NYAA", colorBreadDough
	case "subsplease", "sub":
		return "SUB", colorBreadCrust
	case "thepiratebay", "tpb":
		return "TPB", colorBreadDough
	case "1337x", "1337":
		return "1337", colorBreadCrust
	case "fitgirl", "fitg":
		return "FITG", colorBreadDough
	default:
		if len(tag) > 4 {
			tag = tag[:4]
		}
		return tag, colorDim
	}
}

func FormatSortLabel(sortMode string) string {
	switch strings.ToLower(sortMode) {
	case "seeds":
		return "seeds ↓"
	case "size", "size-desc":
		return "size ↓"
	case "size-asc", "size-up", "smallest":
		return "size ↑"
	case "peers", "leechers", "activity":
		return "peers ↓"
	case "name":
		return "name A–Z"
	case "source":
		return "source A–Z"
	default:
		return sortMode
	}
}

type BrowseViewState struct {
	Width           int
	Height          int
	ActiveCategory  string
	FocusedRegion   string // "sidebar" | "search" | "content" | "detail"
	SearchInput     string
	CursorPos       int
	Results         []engine.TorrentSearchResult
	Cursor          int
	Page            int
	SortMode        string // "seeds" | "size" | "name"
	Searching       bool
	StatusMsg       string
	DetailResult    *engine.TorrentSearchResult
	MinSeeds        int // 0 (all), 1, 5, 20
	SeedingActive   bool
	SeedingName     string
	SeedingProgress engine.BakeProgress
	SeedingMagnet   string
}

func renderRoundedBox(title string, width int, contentLines []string, focused bool) string {
	if width < 20 {
		width = 20
	}
	innerW := width - 2
	contentMaxW := width - 4
	if contentMaxW < 1 {
		contentMaxW = 1
	}

	borderStyle := styleDim
	titleStyle := styleRegular
	if focused {
		borderStyle = styleTitle
		titleStyle = styleSelected
	}

	titleDisplay := ""
	if title != "" {
		titleDisplay = titleStyle.Render(title)
	}

	titleW := lipgloss.Width(title)
	tail := innerW - titleW - 3
	if tail < 0 {
		tail = 0
	}

	var topBorder string
	if title != "" {
		topBorder = borderStyle.Render("╭─ ") + titleDisplay + borderStyle.Render(" "+strings.Repeat("─", tail)+"╮")
	} else {
		topBorder = borderStyle.Render("╭" + strings.Repeat("─", innerW) + "╮")
	}

	var renderedLines []string
	for _, line := range contentLines {
		clipped := lipgloss.NewStyle().MaxWidth(contentMaxW).Render(line)
		lineW := lipgloss.Width(clipped)
		pad := contentMaxW - lineW
		if pad < 0 {
			pad = 0
		}
		renderedLines = append(renderedLines, borderStyle.Render("│ ")+clipped+strings.Repeat(" ", pad)+borderStyle.Render(" │"))
	}

	bottomBorder := borderStyle.Render("╰" + strings.Repeat("─", innerW) + "╯")

	all := append([]string{topBorder}, renderedLines...)
	all = append(all, bottomBorder)
	return strings.Join(all, "\n")
}

func RenderBrowseView(state BrowseViewState) string {
	totalWidth := state.Width
	if totalWidth <= 0 {
		totalWidth = 80
	}
	if totalWidth < 48 {
		totalWidth = 48
	}

	isTwoCol := totalWidth >= 64
	sidebarWidth := 16

	targetContentLines := state.Height - 10
	if targetContentLines < 8 {
		targetContentLines = 8
	}

	// Top Title Bar: clean, focused, no bloated ASCII
	titleBar := styleTitle.Render("🍞 anpan") + styleDim.Render(" · torrent search & browse")
	if state.StatusMsg != "" {
		titleBar += "  " + styleSuccess.Render("✓ "+state.StatusMsg)
	}

	// 1. Sidebar rendering
	var sidebarLines []string
	sidebarLines = append(sidebarLines, "") // small top padding
	for _, cat := range BrowseCategories {
		active := cat.Key == state.ActiveCategory
		if active {
			if state.FocusedRegion == "sidebar" {
				sidebarLines = append(sidebarLines, styleTitle.Render("▌ ")+styleSelected.Render(cat.Label))
			} else {
				sidebarLines = append(sidebarLines, styleDim.Render("▎ ")+styleRegular.Render(cat.Label))
			}
		} else {
			sidebarLines = append(sidebarLines, styleDim.Render("  "+cat.Label))
		}
	}
	sidebarContent := strings.Join(sidebarLines, "\n")

	// 2. Right pane width
	rightWidth := totalWidth
	if isTwoCol {
		rightWidth = totalWidth - sidebarWidth - 2
	}
	if rightWidth < 38 {
		rightWidth = 38
	}

	// 3. Search Box
	searchBoxFocused := state.FocusedRegion == "search"
	searchPrompt := styleDim.Render("❯ ")
	if searchBoxFocused {
		searchPrompt = styleTitle.Render("❯ ")
	}
	searchText := state.SearchInput
	if searchText == "" {
		if searchBoxFocused {
			searchText = styleTitle.Render("█") + styleDim.Render(" Search keywords or paste magnet link…")
		} else {
			searchText = styleDim.Render("Search keywords or paste magnet link…")
		}
	} else {
		if searchBoxFocused {
			searchText = styleRegular.Render(searchText) + styleTitle.Render("█")
		} else {
			searchText = styleRegular.Render(searchText)
		}
	}
	searchBox := renderRoundedBox("Search", rightWidth, []string{searchPrompt + searchText}, searchBoxFocused)

	// 4. Content Panel (Results / Seeding / Details)
	contentBoxFocused := state.FocusedRegion == "content" || state.FocusedRegion == "detail"
	var contentLines []string

	if state.ActiveCategory == "seeding" {
		// Seeding View
		if state.SeedingActive {
			contentLines = append(contentLines, styleTitle.Render("Active Seeding Session"))
			contentLines = append(contentLines, "")
			contentLines = append(contentLines, styleRegular.Render("Name: ")+styleSelected.Render(state.SeedingName))
			if state.SeedingProgress.Speed > 0 {
				contentLines = append(contentLines, styleDim.Render("Upload Speed: ")+styleSuccess.Render(units.FormatSpeed(state.SeedingProgress.Speed)))
			}
			if state.SeedingProgress.DownloadedBytes > 0 {
				contentLines = append(contentLines, styleDim.Render("Uploaded: ")+styleRegular.Render(units.FormatBytes(state.SeedingProgress.DownloadedBytes)))
			}
			if state.SeedingProgress.Connections > 0 {
				contentLines = append(contentLines, styleDim.Render(fmt.Sprintf("Peers: %d", state.SeedingProgress.Connections)))
			}
			if state.SeedingMagnet != "" {
				contentLines = append(contentLines, "")
				magDisplay := units.Truncate(state.SeedingMagnet, rightWidth-10)
				contentLines = append(contentLines, styleDim.Render("Magnet: ")+styleSelected.Render(magDisplay))
			}
		} else {
			contentLines = append(contentLines, styleDim.Render("No active torrent seeding session."))
			contentLines = append(contentLines, "")
			contentLines = append(contentLines, styleRegular.Render("Press ")+styleTitle.Render("^E")+styleRegular.Render(" or drop a file/folder to seed to BitTorrent."))
		}
		for len(contentLines) < targetContentLines {
			contentLines = append(contentLines, "")
		}
	} else if state.FocusedRegion == "detail" && state.DetailResult != nil {
		// Detail View
		r := state.DetailResult
		tag, col := GetSourceBadge(r.Source)
		badge := lipgloss.NewStyle().Foreground(col).Bold(true).Render(tag)
		contentLines = append(contentLines, styleSelected.Render(units.Truncate(r.Title, rightWidth-10))+"  "+badge)
		contentLines = append(contentLines, styleSubtle.Render(strings.Repeat("─", rightWidth-6)))
		contentLines = append(contentLines, "")
		sizeStr := "-"
		if r.SizeBytes > 0 {
			sizeStr = units.FormatBytes(float64(r.SizeBytes))
		}
		contentLines = append(contentLines, styleDim.Render("Size:    ")+styleRegular.Render(sizeStr))
		healthStr := "-"
		if r.Seeders > 0 || r.Leechers > 0 {
			healthStr = fmt.Sprintf("%d seeders · %d leechers", r.Seeders, r.Leechers)
		}
		contentLines = append(contentLines, styleDim.Render("Health:  ")+styleSuccess.Render(healthStr))
		if r.Category != "" {
			contentLines = append(contentLines, styleDim.Render("Category:")+styleDim.Render(" "+r.Category))
		}
		if r.InfoHash != "" {
			contentLines = append(contentLines, styleDim.Render("Hash:    ")+styleSelected.Render(r.InfoHash))
		}
		if r.Magnet != "" {
			contentLines = append(contentLines, styleDim.Render("Magnet:  ")+styleSelected.Render(units.Truncate(r.Magnet, rightWidth-14)))
		}
		contentLines = append(contentLines, "")
		contentLines = append(contentLines, styleTitle.Render("↵/d")+styleRegular.Render(" Download   ")+styleTitle.Render("D")+styleRegular.Render(" Folder   ")+styleTitle.Render("y")+styleRegular.Render(" Copy   ")+styleDim.Render("esc/i Close"))
		for len(contentLines) < targetContentLines {
			contentLines = append(contentLines, "")
		}
	} else {
		// Results Table View
		count := len(state.Results)
		headText := ""
		if state.Searching {
			headText = styleTitle.Render("Searching across 5 indexers…")
		} else if count == 0 {
			if state.SearchInput != "" {
				headText = styleDim.Render(fmt.Sprintf("No results for \"%s\". Try other keywords.", units.Truncate(state.SearchInput, 32)))
			} else {
				headText = styleDim.Render("Type keywords above or browse latest releases")
			}
		} else {
			sortSuffix := ""
			if state.SortMode != "" {
				sortSuffix = fmt.Sprintf(" · sort: %s", FormatSortLabel(state.SortMode))
			}
			filterSuffix := ""
			if state.MinSeeds > 0 {
				filterSuffix = fmt.Sprintf(" · min seeds: %d+", state.MinSeeds)
			}
			headText = styleDim.Render(fmt.Sprintf("%d results across 5 sources%s%s", count, sortSuffix, filterSuffix))
		}
		contentLines = append(contentLines, headText)
		contentLines = append(contentLines, "")

		if count > 0 {
			// Column widths calculation
			contentMaxW := rightWidth - 4
			numW := 3
			if count >= 1000 {
				numW = 4
			}
			sizeW := 9
			seedW := 7
			srcW := 5
			// fixedW: pointer(2) + numW + space(1) + space(1) + sizeW(9) + space(1) + seedW(7) + space(1) + srcW(5)
			fixedW := 2 + numW + 1 + 1 + sizeW + 1 + seedW + 1 + srcW
			nameW := contentMaxW - fixedW
			if nameW < 12 {
				nameW = 12
			}

			// Table Header
			headerRow := fmt.Sprintf("  %*s %-*s %*s %*s %*s",
				numW, "#",
				nameW, "Title",
				sizeW, "Size",
				seedW, "Seeds",
				srcW, "Src",
			)
			contentLines = append(contentLines, styleDim.Render(headerRow))

			// Max visible rows dynamically expands to fill terminal height
			maxVisible := targetContentLines - 4
			if maxVisible < 5 {
				maxVisible = 5
			}

			clampedCursor := state.Cursor
			if clampedCursor < 0 {
				clampedCursor = 0
			}
			if clampedCursor >= count {
				clampedCursor = count - 1
			}

			start := 0
			if clampedCursor >= maxVisible {
				start = clampedCursor - maxVisible + 1
			}
			end := start + maxVisible
			if end > count {
				end = count
			}

			for i := start; i < end; i++ {
				r := state.Results[i]
				isHere := (i == clampedCursor) && (state.FocusedRegion == "content")
				pointer := "  "
				if isHere {
					pointer = styleTitle.Render("❯ ")
				}

				numStr := fmt.Sprintf("%*d", numW, i+1)
				if isHere {
					numStr = styleTitle.Render(numStr)
				} else {
					numStr = styleDim.Render(numStr)
				}

				nameStr := units.Truncate(r.Title, nameW)
				namePad := nameW - lipgloss.Width(nameStr)
				if namePad < 0 {
					namePad = 0
				}
				if isHere {
					nameStr = styleSelected.Render(nameStr) + strings.Repeat(" ", namePad)
				} else {
					nameStr = styleRegular.Render(nameStr) + strings.Repeat(" ", namePad)
				}

				sizeVal := "-"
				if r.SizeBytes > 0 {
					sizeVal = units.FormatBytes(float64(r.SizeBytes))
				}
				sizeStr := fmt.Sprintf("%*s", sizeW, sizeVal)
				if isHere {
					sizeStr = styleRegular.Render(sizeStr)
				} else {
					sizeStr = styleDim.Render(sizeStr)
				}

				seedVal := "-"
				if r.Seeders > 0 || r.Leechers > 0 {
					seedVal = fmt.Sprintf("%d", r.Seeders)
				}
				seedStr := fmt.Sprintf("%*s", seedW, seedVal)
				if r.Seeders > 0 {
					seedStr = styleSuccess.Render(seedStr)
				} else {
					seedStr = styleDim.Render(seedStr)
				}

				badgeTag, badgeColor := GetSourceBadge(r.Source)
				badgeRendered := lipgloss.NewStyle().Foreground(badgeColor).Bold(isHere).Render(fmt.Sprintf("%*s", srcW, badgeTag))

				rowLine := fmt.Sprintf("%s%s %s %s %s %s", pointer, numStr, nameStr, sizeStr, seedStr, badgeRendered)
				contentLines = append(contentLines, rowLine)
			}

			// Scroll / pagination indicator
			currPage := state.Page
			if currPage < 1 {
				currPage = 1
			}
			scrollInfo := fmt.Sprintf("[▲ %d–%d of %d ▼]   ·   page %d   (press \"[\" or \"]\" for prev/next page)", start+1, end, count, currPage)
			contentLines = append(contentLines, styleDim.Render(scrollInfo))
		}

		// Fill remaining lines to keep the box full height
		for len(contentLines) < targetContentLines {
			contentLines = append(contentLines, "")
		}
	}

	panelTitle := "Results"
	if state.ActiveCategory == "seeding" {
		panelTitle = "Seeding"
	} else if state.SearchInput == "" {
		panelTitle = "Latest"
	}
	if len(state.Results) > 0 && state.ActiveCategory != "seeding" {
		currPage := state.Page
		if currPage < 1 {
			currPage = 1
		}
		filterTag := ""
		if state.MinSeeds > 0 {
			filterTag = fmt.Sprintf(" [≥%ds]", state.MinSeeds)
		}
		panelTitle = fmt.Sprintf("%s (%d%s) · Page %d", panelTitle, len(state.Results), filterTag, currPage)
	}

	contentBox := renderRoundedBox(panelTitle, rightWidth, contentLines, contentBoxFocused)

	// Combine right pane
	rightPane := fmt.Sprintf("%s\n%s", searchBox, contentBox)

	// 5. Combine Sidebar & Right Pane
	var mainLayout string
	if isTwoCol {
		sbStyled := lipgloss.NewStyle().Width(sidebarWidth).Render(sidebarContent)
		rpStyled := lipgloss.NewStyle().Width(rightWidth).Render(rightPane)
		mainLayout = lipgloss.JoinHorizontal(lipgloss.Top, sbStyled, " ", rpStyled)
	} else {
		var catPills []string
		for _, cat := range BrowseCategories {
			if cat.Key == state.ActiveCategory {
				catPills = append(catPills, styleTitle.Render("["+cat.Label+"]"))
			} else {
				catPills = append(catPills, styleDim.Render(cat.Label))
			}
		}
		topBar := strings.Join(catPills, "  ")
		mainLayout = fmt.Sprintf("%s\n\n%s", topBar, rightPane)
	}

	// 6. Sleek Minimal Footer Hints (Responsive so it never wraps)
	var footerHints [][2]string
	if totalWidth >= 110 {
		footerHints = [][2]string{
			{"↑↓←→", "move"},
			{"↵/d", "download"},
			{"D", "folder"},
			{"i", "info"},
			{"f", "filter"},
			{"s", "sort"},
			{"y", "copy"},
			{"tab", "switch"},
			{"esc", "back"},
			{"^c", "quit"},
		}
	} else if totalWidth >= 88 {
		footerHints = [][2]string{
			{"↑↓", "move"},
			{"↵", "dl"},
			{"D", "folder"},
			{"i", "info"},
			{"f", "filter"},
			{"s", "sort"},
			{"y", "copy"},
			{"esc", "back"},
		}
	} else {
		footerHints = [][2]string{
			{"↑↓", "move"},
			{"↵", "dl"},
			{"i", "info"},
			{"esc", "back"},
		}
	}
	footerLine := lipgloss.NewStyle().Width(totalWidth).Align(lipgloss.Center).Render(RenderFooterHints(footerHints))

	return fmt.Sprintf("%s\n\n%s\n\n%s", titleBar, mainLayout, footerLine)
}
