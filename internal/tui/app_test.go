package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/KabosuNeko/anpan/internal/core"
	"github.com/KabosuNeko/anpan/internal/engine"
	"github.com/KabosuNeko/anpan/internal/system"
)

func TestAppHistoryNavigation(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")

	system.AddToHistory("https://example.com/first")
	system.AddToHistory("https://example.com/second")

	m := NewModel("0.0.0-test", "", "")
	m.urlInput.SetValue("typed-text")

	// Press Up: should remember typed-text and show most recent history
	upMsg := tea.KeyPressMsg{Code: tea.KeyUp}
	newM, _ := m.Update(upMsg)
	model := newM.(Model)

	if model.urlInput.Value() == "typed-text" {
		t.Errorf("Expected history item after pressing Up, got %s", model.urlInput.Value())
	}
	if model.draftInput != "typed-text" {
		t.Errorf("Expected draftInput to be 'typed-text', got %s", model.draftInput)
	}

	// Press Down: should restore typed-text
	downMsg := tea.KeyPressMsg{Code: tea.KeyDown}
	newM2, _ := model.Update(downMsg)
	model2 := newM2.(Model)

	if model2.urlInput.Value() != "typed-text" {
		t.Errorf("Expected restored draftInput 'typed-text', got %s", model2.urlInput.Value())
	}
	if model2.historyPos != -1 {
		t.Errorf("Expected historyPos to be -1, got %d", model2.historyPos)
	}
}

func TestAppDestPromptNavigation(t *testing.T) {
	m := NewModel("0.0.0-test", "", "")
	m.stage = StageDest
	m.destIndex = 0
	m.isCustomDest = false

	// Press down
	downMsg := tea.KeyPressMsg{Code: tea.KeyDown}
	newM, _ := m.Update(downMsg)
	model := newM.(Model)
	if model.destIndex != 1 {
		t.Errorf("Expected destIndex 1, got %d", model.destIndex)
	}

	// Press 'o' for custom dest
	oMsg := tea.KeyPressMsg{Code: 'o', Text: "o"}
	newM2, _ := model.Update(oMsg)
	model2 := newM2.(Model)
	if !model2.isCustomDest {
		t.Errorf("Expected isCustomDest to be true")
	}

	// Press esc in custom dest: should return to dest options list
	escMsg := tea.KeyPressMsg{Code: tea.KeyEsc}
	newM3, _ := model2.Update(escMsg)
	model3 := newM3.(Model)
	if model3.isCustomDest {
		t.Errorf("Expected isCustomDest to be false after esc")
	}
	if model3.stage != StageDest {
		t.Errorf("Expected stage to still be StageDest, got %s", model3.stage)
	}
}

func TestAppPreferQuality(t *testing.T) {
	m := NewModel("0.0.0-test", "", "")
	m.config.PreferQuality = "audio"
	m.config.AskSaveDir = false

	portions := []engine.Portion{
		{Label: "1080p · MP4", Kind: engine.PortionKindVideo},
		{Label: "Audio · MP3", Kind: engine.PortionKindAudio},
	}
	res := &engine.ProbeResult{
		Meta: engine.VideoMeta{Title: "Test Video"},
	}

	newM, _ := m.Update(probeResultMsg{
		probeResult: res,
		portions:    portions,
	})
	model := newM.(Model)

	if model.stage != StageBaking {
		t.Errorf("Expected StageBaking when preferQuality matched and askSaveDir is false, got %s", model.stage)
	}
	if model.selectedPortion != 1 {
		t.Errorf("Expected audio portion (index 1) to be selected, got %d", model.selectedPortion)
	}
}

func TestAppArchiveMultiFile(t *testing.T) {
	m := NewModel("0.0.0-test", "", "")
	target := &core.TargetInspection{
		Type: core.TargetArchive,
		URL:  "https://coomer.su/onlyfans/user/test/post/123",
		ArchivePost: &engine.ArchivePost{
			Title: "Test Archive Post",
			Files: []engine.ArchiveFile{
				{Name: "file1.jpg", URL: "https://example.com/1.jpg"},
				{Name: "file2.jpg", URL: "https://example.com/2.jpg"},
			},
		},
	}

	newM, _ := m.Update(inspectMsg{target: target})
	model := newM.(Model)

	if model.stage != StageSelecting {
		t.Errorf("Expected StageSelecting for multi-file archive, got %s", model.stage)
	}
	if len(model.portions) != 3 {
		t.Fatalf("Expected 3 portions (all + 2 files), got %d", len(model.portions))
	}

	// Select portion 0 (all files)
	enterMsg := tea.KeyPressMsg{Code: tea.KeyEnter}
	model.config.AskSaveDir = false
	newM2, _ := model.Update(enterMsg)
	model2 := newM2.(Model)

	if model2.stage != StageBaking {
		t.Errorf("Expected StageBaking, got %s", model2.stage)
	}
	if len(model2.selectedArchiveFiles) != 2 {
		t.Errorf("Expected 2 archive files selected, got %d", len(model2.selectedArchiveFiles))
	}
}

func TestAppCtrlFSearch(t *testing.T) {
	m := NewModel("0.0.0-test", "", "")

	// Press ctrl+f from StageInput
	ctrlFMsg := tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl}
	newM, cmd := m.Update(ctrlFMsg)
	model := newM.(Model)

	if model.stage != StageSearch {
		t.Fatalf("Expected stage StageSearch after ctrl+f, got %s", model.stage)
	}
	if model.browseRegion != "search" {
		t.Errorf("Expected browseRegion 'search', got %s", model.browseRegion)
	}
	if cmd == nil {
		t.Errorf("Expected batch cmd for search spinner and search query")
	}

	// Tab should switch region to content
	tabMsg := tea.KeyPressMsg{Code: tea.KeyTab}
	newM2, _ := model.Update(tabMsg)
	model2 := newM2.(Model)
	if model2.browseRegion != "content" {
		t.Errorf("Expected browseRegion 'content' after Tab, got %s", model2.browseRegion)
	}

	// Tab again should switch region to sidebar
	newM3, _ := model2.Update(tabMsg)
	model3 := newM3.(Model)
	if model3.browseRegion != "sidebar" {
		t.Errorf("Expected browseRegion 'sidebar' after second Tab, got %s", model3.browseRegion)
	}

	// Down in sidebar should change category
	downMsg := tea.KeyPressMsg{Code: tea.KeyDown}
	newM4, _ := model3.Update(downMsg)
	model4 := newM4.(Model)
	if model4.browseCategory != "anime" {
		t.Errorf("Expected category 'anime' after down in sidebar, got %s", model4.browseCategory)
	}

	// Escape from content should return to StageInput
	model4.browseRegion = "content"
	escMsg := tea.KeyPressMsg{Code: tea.KeyEsc}
	newM5, _ := model4.Update(escMsg)
	model5 := newM5.(Model)
	if model5.stage != StageInput {
		t.Errorf("Expected StageInput after Esc from content, got %s", model5.stage)
	}
}

func TestAppCtrlESeed(t *testing.T) {
	m := NewModel("0.0.0-test", "", "")

	// Press ctrl+e from StageInput
	ctrlEMsg := tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl}
	newM, _ := m.Update(ctrlEMsg)
	model := newM.(Model)

	if model.stage != StageSeedPrompt {
		t.Fatalf("Expected stage StageSeedPrompt after ctrl+e, got %s", model.stage)
	}

	// Press Esc: should return to StageInput
	escMsg := tea.KeyPressMsg{Code: tea.KeyEsc}
	newM2, _ := model.Update(escMsg)
	model2 := newM2.(Model)
	if model2.stage != StageInput {
		t.Errorf("Expected StageInput after Esc from seed prompt, got %s", model2.stage)
	}
}

func TestAppSearchActionsAndSort(t *testing.T) {
	m := NewModel("0.0.0-test", "", "")
	m.stage = StageSearch
	m.browseRegion = "content"
	m.browseResults = []engine.TorrentSearchResult{
		{
			Title:     "B Movie (2024)",
			SizeBytes: 1000,
			Seeders:   10,
			Magnet:    "magnet:?xt=urn:btih:1111",
			Source:    "yts",
		},
		{
			Title:     "A Movie (2024)",
			SizeBytes: 5000,
			Seeders:   50,
			Magnet:    "magnet:?xt=urn:btih:2222",
			Source:    "yts",
		},
	}
	m.browseCursor = 0
	m.config.AskSaveDir = false

	// Test sort key 's': seeds -> size -> size-asc -> peers -> name -> source -> seeds
	sMsg := tea.KeyPressMsg{Code: 's', Text: "s"}
	newM, _ := m.Update(sMsg)
	model := newM.(Model)
	if model.browseSort != "size" {
		t.Errorf("Expected browseSort 'size', got %s", model.browseSort)
	}
	// With sort by size desc, item with 5000 bytes should be first
	if model.browseResults[0].SizeBytes != 5000 {
		t.Errorf("Expected largest result first after size sort, got %d", model.browseResults[0].SizeBytes)
	}

	// Press 's' again -> size-asc
	newM, _ = model.Update(sMsg)
	model = newM.(Model)
	if model.browseSort != "size-asc" {
		t.Errorf("Expected browseSort 'size-asc', got %s", model.browseSort)
	}
	if model.browseResults[0].SizeBytes != 1000 {
		t.Errorf("Expected smallest result first after size-asc sort, got %d", model.browseResults[0].SizeBytes)
	}

	// Press 's' again -> peers
	newM, _ = model.Update(sMsg)
	model = newM.(Model)
	if model.browseSort != "peers" {
		t.Errorf("Expected browseSort 'peers', got %s", model.browseSort)
	}

	// Press 's' again -> name
	newM, _ = model.Update(sMsg)
	model = newM.(Model)
	if model.browseSort != "name" {
		t.Errorf("Expected browseSort 'name', got %s", model.browseSort)
	}

	// Press 's' again -> source
	newM, _ = model.Update(sMsg)
	model = newM.(Model)
	if model.browseSort != "source" {
		t.Errorf("Expected browseSort 'source', got %s", model.browseSort)
	}

	// Press 's' again -> seeds
	newM, _ = model.Update(sMsg)
	model = newM.(Model)
	if model.browseSort != "seeds" {
		t.Errorf("Expected browseSort 'seeds', got %s", model.browseSort)
	}

	// Test download key 'd': should start bake
	dMsg := tea.KeyPressMsg{Code: 'd', Text: "d"}
	newM2, _ := model.Update(dMsg)
	model2 := newM2.(Model)
	if model2.stage != StageBaking {
		t.Errorf("Expected StageBaking after pressing 'd' on result, got %s", model2.stage)
	}
	if model2.target == nil || model2.target.Name != "A Movie (2024)" {
		t.Errorf("Expected target name 'A Movie (2024)', got %+v", model2.target)
	}
}

func TestAppInspectTargetSearchAndSeed(t *testing.T) {
	m := NewModel("0.0.0-test", "", "")

	// 1. Inspect TargetSearch
	searchTarget := &core.TargetInspection{
		Type:        core.TargetSearch,
		SearchQuery: "cyberpunk",
	}
	newM, _ := m.Update(inspectMsg{target: searchTarget})
	model := newM.(Model)

	if model.stage != StageSearch {
		t.Errorf("Expected StageSearch for TargetSearch, got %s", model.stage)
	}
	if model.browseInput.Value() != "cyberpunk" {
		t.Errorf("Expected browseInput 'cyberpunk', got %s", model.browseInput.Value())
	}

	// 2. Inspect TargetSeed
	seedTarget := &core.TargetInspection{
		Type:   core.TargetSeed,
		Target: "/path/to/my/folder",
	}
	newM2, _ := m.Update(inspectMsg{target: seedTarget})
	model2 := newM2.(Model)

	if model2.stage != StageSeedPrompt {
		t.Errorf("Expected StageSeedPrompt for TargetSeed, got %s", model2.stage)
	}
	if model2.seedInput.Value() != "/path/to/my/folder" {
		t.Errorf("Expected seedInput '/path/to/my/folder', got %s", model2.seedInput.Value())
	}
}

func TestAppSearchViewNoMascot(t *testing.T) {
	m := NewModel("0.0.0-test", "", "")
	m.stage = StageSearch
	m.width = 80
	m.height = 24

	view := m.View()
	content := view.Content

	// Verify mascot is NOT rendered in search mode
	if strings.Contains(content, "(.. )") || strings.Contains(content, "feed a link, bake a file") {
		t.Errorf("Expected mascot and tagline to be omitted in StageSearch")
	}

	// Verify clean header and search UI are rendered
	if !strings.Contains(content, "anpan") || !strings.Contains(content, "torrent search & browse") {
		t.Errorf("Expected sleek header in StageSearch, got: %s", content)
	}
	if !strings.Contains(content, "Search") {
		t.Errorf("Expected Search box in StageSearch")
	}
}

func TestAppSearchPagination(t *testing.T) {
	m := NewModel("0.0.0-test", "", "")
	m.stage = StageSearch
	m.browseRegion = "content"
	m.browsePage = 1

	for i := 0; i < 30; i++ {
		m.browseResults = append(m.browseResults, engine.TorrentSearchResult{
			Title:   fmt.Sprintf("Item %d", i),
			Seeders: 100 - i,
		})
	}

	// Next page key ']'
	nextMsg := tea.KeyPressMsg{Code: ']', Text: "]"}
	newM, cmd := m.Update(nextMsg)
	model := newM.(Model)
	if model.browsePage != 2 {
		t.Errorf("Expected browsePage 2 after ']', got %d", model.browsePage)
	}
	if cmd == nil {
		t.Errorf("Expected search cmd triggered for page 2")
	}

	// Prev page key '['
	prevMsg := tea.KeyPressMsg{Code: '[', Text: "["}
	newM2, cmd2 := model.Update(prevMsg)
	model2 := newM2.(Model)
	if model2.browsePage != 1 {
		t.Errorf("Expected browsePage 1 after '[', got %d", model2.browsePage)
	}
	if cmd2 == nil {
		t.Errorf("Expected search cmd triggered for page 1")
	}

	// Jump down 'pgdown'
	pgDownMsg := tea.KeyPressMsg{Code: tea.KeyPgDown}
	newM3, _ := model2.Update(pgDownMsg)
	model3 := newM3.(Model)
	if model3.browseCursor != 10 {
		t.Errorf("Expected cursor 10 after pgdown, got %d", model3.browseCursor)
	}

	// Jump end 'end'
	endMsg := tea.KeyPressMsg{Code: tea.KeyEnd}
	newM4, _ := model3.Update(endMsg)
	model4 := newM4.(Model)
	if model4.browseCursor != len(model4.browseResults)-1 {
		t.Errorf("Expected cursor at end (%d), got %d", len(model4.browseResults)-1, model4.browseCursor)
	}

	// Jump home 'home'
	homeMsg := tea.KeyPressMsg{Code: tea.KeyHome}
	newM5, _ := model4.Update(homeMsg)
	model5 := newM5.(Model)
	if model5.browseCursor != 0 {
		t.Errorf("Expected cursor 0 after home, got %d", model5.browseCursor)
	}
}

func TestAppSearchGamesCategory(t *testing.T) {
	m := NewModel("0.0.0-test", "", "")
	m.stage = StageSearch
	m.browseRegion = "sidebar"
	m.browseCategory = "all"

	// Press down 4 times: all -> anime -> movies -> tv -> games
	downMsg := tea.KeyPressMsg{Code: tea.KeyDown}
	cur := m
	for i := 0; i < 4; i++ {
		newM, _ := cur.Update(downMsg)
		cur = newM.(Model)
	}

	if cur.browseCategory != "games" {
		t.Errorf("Expected category 'games' after 4 down presses, got %s", cur.browseCategory)
	}
}

func TestAppInspectorDetail(t *testing.T) {
	m := NewModel("0.0.0-test", "", "")
	m.stage = StageSearch
	m.browseRegion = "content"
	m.browseResults = []engine.TorrentSearchResult{
		{Title: "Ubuntu 24.04 ISO", Magnet: "magnet:?xt=urn:btih:1111", Seeders: 100},
		{Title: "Debian 12 ISO", Magnet: "magnet:?xt=urn:btih:2222", Seeders: 50},
	}
	m.browseCursor = 0

	// 1. Press 'i': should enter detail view
	iMsg := tea.KeyPressMsg{Code: 'i', Text: "i"}
	newM, _ := m.Update(iMsg)
	m1 := newM.(Model)
	if m1.browseRegion != "detail" || m1.browseDetail == nil {
		t.Fatalf("Expected detail view, got region %s, detail: %v", m1.browseRegion, m1.browseDetail)
	}
	if m1.browseDetail.Title != "Ubuntu 24.04 ISO" {
		t.Errorf("Expected Ubuntu detail, got %s", m1.browseDetail.Title)
	}

	// 2. Press 'i' again: should return to content
	newM2, _ := m1.Update(iMsg)
	m2 := newM2.(Model)
	if m2.browseRegion != "content" || m2.browseDetail != nil {
		t.Errorf("Expected content view after toggle, got %s", m2.browseRegion)
	}

	// 3. Press Space: should enter detail view
	spaceMsg := tea.KeyPressMsg{Code: ' ', Text: " "}
	newM3, _ := m2.Update(spaceMsg)
	m3 := newM3.(Model)
	if m3.browseRegion != "detail" || m3.browseDetail == nil {
		t.Errorf("Expected detail view after space, got %s", m3.browseRegion)
	}

	// 4. Press Esc: should return to content
	escMsg := tea.KeyPressMsg{Code: tea.KeyEsc}
	newM4, _ := m3.Update(escMsg)
	m4 := newM4.(Model)
	if m4.browseRegion != "content" || m4.browseDetail != nil {
		t.Errorf("Expected content view after Esc, got %s", m4.browseRegion)
	}
}

func TestAppMinSeedsFilter(t *testing.T) {
	m := NewModel("0.0.0-test", "", "")
	m.stage = StageSearch
	m.browseRegion = "content"
	m.browseResults = []engine.TorrentSearchResult{
		{Title: "Dead Torrent", Seeders: 0},
		{Title: "Low Seeds Torrent", Seeders: 3},
		{Title: "Medium Seeds Torrent", Seeders: 10},
		{Title: "High Seeds Torrent", Seeders: 25},
	}

	if len(m.getFilteredBrowseResults()) != 4 {
		t.Fatalf("Expected 4 results initially, got %d", len(m.getFilteredBrowseResults()))
	}

	fMsg := tea.KeyPressMsg{Code: 'f', Text: "f"}

	// 1st press: seeds >= 1 (3 items)
	newM1, _ := m.Update(fMsg)
	m1 := newM1.(Model)
	if m1.browseMinSeeds != 1 {
		t.Errorf("Expected browseMinSeeds 1, got %d", m1.browseMinSeeds)
	}
	if len(m1.getFilteredBrowseResults()) != 3 {
		t.Errorf("Expected 3 filtered results, got %d", len(m1.getFilteredBrowseResults()))
	}

	// 2nd press: seeds >= 5 (2 items)
	newM2, _ := m1.Update(fMsg)
	m2 := newM2.(Model)
	if m2.browseMinSeeds != 5 {
		t.Errorf("Expected browseMinSeeds 5, got %d", m2.browseMinSeeds)
	}
	if len(m2.getFilteredBrowseResults()) != 2 {
		t.Errorf("Expected 2 filtered results, got %d", len(m2.getFilteredBrowseResults()))
	}

	// 3rd press: seeds >= 20 (1 item)
	newM3, _ := m2.Update(fMsg)
	m3 := newM3.(Model)
	if m3.browseMinSeeds != 20 {
		t.Errorf("Expected browseMinSeeds 20, got %d", m3.browseMinSeeds)
	}
	if len(m3.getFilteredBrowseResults()) != 1 {
		t.Errorf("Expected 1 filtered result, got %d", len(m3.getFilteredBrowseResults()))
	}

	// 4th press: seeds >= 0 (all 4 items)
	newM4, _ := m3.Update(fMsg)
	m4 := newM4.(Model)
	if m4.browseMinSeeds != 0 {
		t.Errorf("Expected browseMinSeeds 0, got %d", m4.browseMinSeeds)
	}
	if len(m4.getFilteredBrowseResults()) != 4 {
		t.Errorf("Expected 4 results reset, got %d", len(m4.getFilteredBrowseResults()))
	}
}

func TestAppSearchHistoryNavigation(t *testing.T) {
	m := NewModel("0.0.0-test", "", "")
	m.stage = StageSearch
	m.browseRegion = "search"
	m.searchHistory = []string{"ubuntu", "debian"}
	m.browseInput.SetValue("arch")

	upMsg := tea.KeyPressMsg{Code: tea.KeyUp}
	downMsg := tea.KeyPressMsg{Code: tea.KeyDown}

	// Press Up -> "ubuntu"
	newM1, _ := m.Update(upMsg)
	m1 := newM1.(Model)
	if m1.browseInput.Value() != "ubuntu" {
		t.Errorf("Expected 'ubuntu', got %s", m1.browseInput.Value())
	}
	if m1.draftSearch != "arch" {
		t.Errorf("Expected draftSearch 'arch', got %s", m1.draftSearch)
	}

	// Press Up -> "debian"
	newM2, _ := m1.Update(upMsg)
	m2 := newM2.(Model)
	if m2.browseInput.Value() != "debian" {
		t.Errorf("Expected 'debian', got %s", m2.browseInput.Value())
	}

	// Press Down -> "ubuntu"
	newM3, _ := m2.Update(downMsg)
	m3 := newM3.(Model)
	if m3.browseInput.Value() != "ubuntu" {
		t.Errorf("Expected 'ubuntu', got %s", m3.browseInput.Value())
	}

	// Press Down -> "arch" (restored draft)
	newM4, _ := m3.Update(downMsg)
	m4 := newM4.(Model)
	if m4.browseInput.Value() != "arch" {
		t.Errorf("Expected 'arch', got %s", m4.browseInput.Value())
	}

	// Submit search: should prepend to history
	m4.browseInput.SetValue("fedora")
	enterMsg := tea.KeyPressMsg{Code: tea.KeyEnter}
	newM5, _ := m4.Update(enterMsg)
	m5 := newM5.(Model)
	if len(m5.searchHistory) != 3 || m5.searchHistory[0] != "fedora" {
		t.Errorf("Expected 'fedora' at head of searchHistory, got %v", m5.searchHistory)
	}
}

func TestAppCustomDestD(t *testing.T) {
	m := NewModel("0.0.0-test", "", "")
	m.stage = StageSearch
	m.browseRegion = "content"
	m.browseResults = []engine.TorrentSearchResult{
		{Title: "Arch Linux 2026.09", Magnet: "magnet:?xt=urn:btih:3333", Source: "nyaa"},
	}
	m.browseCursor = 0

	dMsg := tea.KeyPressMsg{Code: 'D', Text: "D"}
	newM, _ := m.Update(dMsg)
	m1 := newM.(Model)

	if m1.stage != StageDest {
		t.Errorf("Expected StageDest after Shift+D, got %s", m1.stage)
	}
	if m1.destTargetTitle != "Arch Linux 2026.09" {
		t.Errorf("Expected destTargetTitle 'Arch Linux 2026.09', got %s", m1.destTargetTitle)
	}
}

func TestAppSearchPageKeysPN(t *testing.T) {
	m := NewModel("0.0.0-test", "", "")
	m.stage = StageSearch
	m.browseRegion = "content"
	m.browseCategory = "all"
	m.browseInput.SetValue("ubuntu")
	m.browsePage = 1

	// 'n' moves to page 2 and triggers search
	nMsg := tea.KeyPressMsg{Code: 'n', Text: "n"}
	newM1, cmd1 := m.Update(nMsg)
	m1 := newM1.(Model)
	if m1.browsePage != 2 {
		t.Errorf("Expected browsePage 2 after 'n', got %d", m1.browsePage)
	}
	if cmd1 == nil {
		t.Errorf("Expected command returned for next page search")
	}

	// 'p' moves back to page 1
	pMsg := tea.KeyPressMsg{Code: 'p', Text: "p"}
	newM2, cmd2 := m1.Update(pMsg)
	m2 := newM2.(Model)
	if m2.browsePage != 1 {
		t.Errorf("Expected browsePage 1 after 'p', got %d", m2.browsePage)
	}
	if cmd2 == nil {
		t.Errorf("Expected command returned for prev page search")
	}

	// 'p' at page 1 does not decrement further
	newM3, cmd3 := m2.Update(pMsg)
	m3 := newM3.(Model)
	if m3.browsePage != 1 {
		t.Errorf("Expected browsePage to stay at 1, got %d", m3.browsePage)
	}
	if cmd3 != nil {
		t.Errorf("Expected nil command when already on page 1")
	}
}

func TestAppTabbedSettingsNavigation(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)

	m := NewModel("0.0.0-test", "", "")

	// 1. Open settings
	ctrlSMsg := tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl, Text: "ctrl+s"}
	newM, _ := m.Update(ctrlSMsg)
	m1 := newM.(Model)

	if m1.stage != StageSettings {
		t.Fatalf("Expected StageSettings, got %s", m1.stage)
	}
	if m1.settingsTab != 0 || m1.settingsIndex != 0 {
		t.Errorf("Expected tab 0 index 0, got tab %d index %d", m1.settingsTab, m1.settingsIndex)
	}

	// 2. Press Tab -> Tab 1 (Video)
	tabMsg := tea.KeyPressMsg{Code: tea.KeyTab}
	newM2, _ := m1.Update(tabMsg)
	m2 := newM2.(Model)
	if m2.settingsTab != 1 {
		t.Errorf("Expected tab 1 after Tab, got %d", m2.settingsTab)
	}

	// 3. Press ']' -> Tab 2 (Audio)
	bracketNextMsg := tea.KeyPressMsg{Code: ']', Text: "]"}
	newM3, _ := m2.Update(bracketNextMsg)
	m3 := newM3.(Model)
	if m3.settingsTab != 2 {
		t.Errorf("Expected tab 2 after ']', got %d", m3.settingsTab)
	}

	// 4. Press '4' -> Tab 3 (Torrent)
	num4Msg := tea.KeyPressMsg{Code: '4', Text: "4"}
	newM4, _ := m3.Update(num4Msg)
	m4 := newM4.(Model)
	if m4.settingsTab != 3 {
		t.Errorf("Expected tab 3 after '4', got %d", m4.settingsTab)
	}

	// 5. Press '[' -> Tab 2 (Audio)
	bracketPrevMsg := tea.KeyPressMsg{Code: '[', Text: "["}
	newM5, _ := m4.Update(bracketPrevMsg)
	m5 := newM5.(Model)
	if m5.settingsTab != 2 {
		t.Errorf("Expected tab 2 after '[', got %d", m5.settingsTab)
	}

	// 6. Press Shift+Tab -> Tab 1 (Video)
	shiftTabMsg := tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	newM6, _ := m5.Update(shiftTabMsg)
	m6 := newM6.(Model)
	if m6.settingsTab != 1 {
		t.Errorf("Expected tab 1 after Shift+Tab, got %d", m6.settingsTab)
	}

	// 7. Navigate down to item 3 (subtitles: 0:container, 1:codec, 2:quality, 3:subtitles)
	downMsg := tea.KeyPressMsg{Code: tea.KeyDown}
	cur := m6
	for i := 0; i < 3; i++ {
		next, _ := cur.Update(downMsg)
		cur = next.(Model)
	}
	if cur.settingsIndex != 3 {
		t.Errorf("Expected settingsIndex 3, got %d", cur.settingsIndex)
	}

	// 8. Cycle setting right: subtitles ("off" -> "embed")
	rightMsg := tea.KeyPressMsg{Code: tea.KeyRight}
	newM7, _ := cur.Update(rightMsg)
	m7 := newM7.(Model)
	if m7.config.Subtitles != "embed" {
		t.Errorf("Expected subtitles 'embed', got %s", m7.config.Subtitles)
	}

	// 9. Press Esc -> StageInput
	escMsg := tea.KeyPressMsg{Code: tea.KeyEsc}
	newM8, _ := m7.Update(escMsg)
	m8 := newM8.(Model)
	if m8.stage != StageInput {
		t.Errorf("Expected stage StageInput after esc, got %s", m8.stage)
	}
}


