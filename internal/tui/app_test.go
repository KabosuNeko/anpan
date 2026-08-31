package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/KabosuNeko/anpan/internal/core"
	"github.com/KabosuNeko/anpan/internal/engine"
	"github.com/KabosuNeko/anpan/internal/system"
)

func TestAppHistoryNavigation(t *testing.T) {
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
