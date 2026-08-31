package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/KabosuNeko/anpan/internal/core"
	"github.com/KabosuNeko/anpan/internal/engine"
	"github.com/KabosuNeko/anpan/internal/system"
	"github.com/KabosuNeko/anpan/internal/units"
)

type StageName string

const (
	StageInput          StageName = "input"
	StageProbing        StageName = "probing"
	StageDirectPrompt   StageName = "direct_prompt"
	StageTorrentPrompt  StageName = "torrent_prompt"
	StagePlaylistPrompt StageName = "playlist_prompt"
	StageSelecting      StageName = "selecting"
	StageDest           StageName = "dest"
	StageBaking         StageName = "baking"
	StageBaked          StageName = "baked"
	StageError          StageName = "error"
	StageSettings       StageName = "settings"
)

var stageHints = map[StageName][][2]string{
	StageInput: {
		{"↑↓", "history"},
		{"↵", "bake"},
		{"^s", "settings"},
		{"^c", "quit"},
	},
	StageProbing: {
		{"esc", "cancel"},
		{"^c", "quit"},
	},
	StageDirectPrompt: {
		{"↑↓", "choose"},
		{"↵", "download"},
		{"esc", "cancel"},
		{"^c", "quit"},
	},
	StageTorrentPrompt: {
		{"↑↓", "choose"},
		{"↵", "download"},
		{"esc", "cancel"},
		{"^c", "quit"},
	},
	StagePlaylistPrompt: {
		{"↑↓", "choose"},
		{"↵", "select"},
		{"esc", "cancel"},
		{"^c", "quit"},
	},
	StageSelecting: {
		{"↑↓", "choose"},
		{"↵", "download"},
		{"esc", "back"},
		{"^c", "quit"},
	},
	StageDest: {
		{"↑↓", "choose"},
		{"↵", "confirm"},
		{"D/V/C", "quick folder"},
		{"esc", "back"},
		{"^c", "quit"},
	},
	StageBaking: {
		{"esc", "cancel"},
		{"^c", "quit"},
	},
	StageBaked: {
		{"↵", "again"},
		{"^s", "settings"},
		{"^c", "quit"},
	},
	StageError: {
		{"↵", "retry"},
		{"^c", "quit"},
	},
	StageSettings: {
		{"↑↓", "select"},
		{"↵", "edit/toggle"},
		{"⇄", "preset"},
		{"esc", "close"},
	},
}

type errMsg error

type inspectMsg struct {
	target *core.TargetInspection
	err    error
}

type probeResultMsg struct {
	probeResult *engine.ProbeResult
	portions    []engine.Portion
	err         error
}

type probePlaylistMsg struct {
	meta *engine.PlaylistMeta
	err  error
}

type bakeProgressMsg engine.BakeProgress

type bakeProcessingMsg struct{}

type updateCheckMsg struct {
	latestVersion string
}

type bakeDoneMsg struct {
	path string
	err  error
}

type ProgramBridge struct {
	mu sync.Mutex
	p  *tea.Program
}

func (b *ProgramBridge) SetProgram(p *tea.Program) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.p = p
}

func (b *ProgramBridge) Send(msg tea.Msg) {
	b.mu.Lock()
	p := b.p
	b.mu.Unlock()
	if p != nil {
		p.Send(msg)
	}
}

type Model struct {
	version        string
	latestVersion  string
	initialURL     string
	initialOutDir  string
	width          int
	height         int
	stage          StageName
	statusText     string
	errText        string
	config         system.AnpanConfig

	bridge *ProgramBridge

	// Input stage & history
	urlInput     textinput.Model
	clipboardURL string
	history      []string
	historyPos   int
	draftInput   string

	// Probing stage
	spinner spinner.Model

	// Target & Extracted data
	target               *core.TargetInspection
	probeResult          *engine.ProbeResult
	playlistMeta         *engine.PlaylistMeta
	isPlaylistMode       bool
	archivePost          *engine.ArchivePost
	selectedArchiveFiles []engine.ArchiveFile
	portions             []engine.Portion
	selectedPortion      int
	promptChoice         int // 0: download/full, 1: cancel/single

	// Dest stage
	destIndex       int
	isCustomDest    bool
	destTargetTitle string
	destTargetSub   string
	destInput       textinput.Model
	chosenDest      string

	// Baking stage
	bakingProgress engine.BakeProgress
	processing     bool
	resultFilePath string

	// Settings stage
	settingsIndex int
	editingDir    bool
	settingsDir   textinput.Model

	// Final outcome
	FinalPath string

	ctx       context.Context
	cancelCtx context.CancelFunc
}

func (m Model) SetProgram(p *tea.Program) {
	if m.bridge != nil {
		m.bridge.SetProgram(p)
	}
}

func NewModel(version, initialURL, initialOutDir string) Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Focus()
	ti.CharLimit = 2048

	di := textinput.New()
	di.Prompt = ""
	di.CharLimit = 512

	sdi := textinput.New()
	sdi.Prompt = ""
	sdi.CharLimit = 512

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styleRegular

	ctx, cancel := context.WithCancel(context.Background())

	m := Model{
		version:       version,
		initialURL:    initialURL,
		initialOutDir: initialOutDir,
		stage:         StageInput,
		config:        system.LoadConfig(),
		bridge:        &ProgramBridge{},
		history:       system.LoadHistory(),
		historyPos:    -1,
		urlInput:      ti,
		destInput:     di,
		settingsDir:   sdi,
		spinner:       s,
		width:         80,
		height:        24,
		ctx:           ctx,
		cancelCtx:     cancel,
	}

	clip := system.ReadClipboard()
	if clip != "" && core.IsLikelyTarget(clip) {
		m.clipboardURL = clip
		m.urlInput.Placeholder = clip + "  ⇥ paste"
	} else {
		m.urlInput.Placeholder = "https://... or magnet:?..."
	}

	if initialURL != "" {
		m.urlInput.SetValue(initialURL)
		m.stage = StageProbing
		m.statusText = "probing target…"
	}

	return m
}

func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, textinput.Blink)
	if m.stage == StageProbing {
		cmds = append(cmds, m.spinner.Tick, m.startInspect(m.urlInput.Value()))
	}
	cmds = append(cmds, func() tea.Msg {
		check := system.CheckUpdate(context.Background(), m.version, nil)
		if check != nil && check.UpdateAvailable {
			return updateCheckMsg{latestVersion: check.LatestVersion}
		}
		return nil
	})
	return tea.Batch(cmds...)
}

func (m Model) startInspect(raw string) tea.Cmd {
	return func() tea.Msg {
		target, err := core.InspectTarget(m.ctx, raw)
		return inspectMsg{target: target, err: err}
	}
}

func (m Model) startProbeVideo(target *core.TargetInspection) tea.Cmd {
	return func() tea.Msg {
		ytdlp, err := engine.EnsureYtDlpBinary(m.ctx, nil)
		if err != nil {
			return probeResultMsg{err: err}
		}

		// If playlist url
		if core.IsPlaylistURL(target.CleanURL) {
			plMeta, plErr := engine.ProbePlaylist(m.ctx, ytdlp, target.CleanURL)
			if plErr == nil && plMeta != nil {
				return probePlaylistMsg{meta: plMeta}
			}
		}

		res, err := engine.ProbeVideo(m.ctx, ytdlp, target.CleanURL)
		if err != nil {
			return probeResultMsg{err: err}
		}
		opts := &engine.ExtractPortionsOptions{
			VideoContainer: m.config.VideoContainer,
			AudioFormat:    m.config.AudioFormat,
			EmbedMetadata:  &m.config.EmbedMetadata,
		}
		portions := engine.ExtractPortions(res.Meta, opts)
		return probeResultMsg{probeResult: res, portions: portions}
	}
}

func (m Model) startBake() tea.Cmd {
	return func() tea.Msg {
		outDir := m.chosenDest
		if outDir == "" {
			outDir = m.config.OutDir
		}

		handlers := engine.BakeHandlers{
			OnProgress: func(p engine.BakeProgress) {
				m.bridge.Send(bakeProgressMsg(p))
			},
			OnProcessing: func() {
				m.bridge.Send(bakeProcessingMsg{})
			},
		}

		if m.target.Type == core.TargetDirect {
			aria2c, err := engine.FindAria2c()
			if err != nil {
				return bakeDoneMsg{err: fmt.Errorf("aria2c required for direct downloads: %w", err)}
			}
			path, err := engine.BakeDirectDownload(m.ctx, engine.DirectDownloadOptions{
				Aria2cBin:   aria2c,
				URL:         m.target.URL,
				Filename:    m.target.Filename,
				OutputDir:   outDir,
				Connections: m.config.Connections,
			}, handlers)
			return bakeDoneMsg{path: path, err: err}
		}

		if m.target.Type == core.TargetTorrent {
			aria2c, err := engine.FindAria2c()
			if err != nil {
				return bakeDoneMsg{err: fmt.Errorf("aria2c required for torrents: %w", err)}
			}
			path, err := engine.BakeTorrentDownload(m.ctx, engine.TorrentDownloadOptions{
				Aria2cBin: aria2c,
				Target:    m.target.Target,
				OutputDir: outDir,
			}, handlers)
			return bakeDoneMsg{path: path, err: err}
		}

		if m.target.Type == core.TargetArchive {
			aria2c, err := engine.FindAria2c()
			if err != nil {
				return bakeDoneMsg{err: fmt.Errorf("aria2c required for archive batch: %w", err)}
			}
			filesToDownload := m.selectedArchiveFiles
			if len(filesToDownload) == 0 && m.archivePost != nil {
				filesToDownload = m.archivePost.Files
			}
			var items []engine.BatchItem
			for _, f := range filesToDownload {
				items = append(items, engine.BatchItem{
					URL:      f.URL,
					Mirrors:  f.Mirrors,
					Filename: f.Name,
				})
			}
			folder := outDir
			if len(filesToDownload) > 1 && m.archivePost != nil {
				folder = filepath.Join(outDir, m.archivePost.Title)
			}
			path, err := engine.BakeBatchDownload(m.ctx, engine.BatchDownloadOptions{
				Aria2cBin:   aria2c,
				Items:       items,
				OutputDir:   folder,
				Connections: m.config.Connections,
			}, handlers)
			return bakeDoneMsg{path: path, err: err}
		}

		// Video / Playlist bake
		ytdlp, err := engine.EnsureYtDlpBinary(m.ctx, nil)
		if err != nil {
			return bakeDoneMsg{err: err}
		}
		ffmpegDir := engine.FindFfmpeg()
		portion := m.portions[m.selectedPortion]

		cachedPath := ""
		if m.probeResult != nil {
			cachedPath = m.probeResult.CachedJSONPath
		}

		var aria2cArgs []string
		if m.config.Aria2c {
			if ariaBin, aErr := engine.FindAria2c(); aErr == nil {
				aria2cArgs = engine.BuildAria2cArgs(ariaBin, m.config.Connections)
			}
		}

		path, err := engine.BakeVideo(m.ctx, engine.BakeVideoOptions{
			YtdlpBin:       ytdlp,
			FfmpegLocation: ffmpegDir,
			Aria2cArgs:     aria2cArgs,
			URL:            m.target.CleanURL,
			CachedJSONPath: cachedPath,
			Portion:        portion,
			OutputDir:      outDir,
			TimeRange:      m.target.TimeRange,
			IsPlaylist:     m.isPlaylistMode,
			CookiesBrowser: m.config.CookiesBrowser,
			Subtitles:      m.config.Subtitles,
			SubLangs:       m.config.SubLangs,
			SponsorBlock:   m.config.SponsorBlock,
			WriteThumbnail: m.config.WriteThumbnail,
		}, handlers)
		return bakeDoneMsg{path: path, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		panelWidth := 64
		if m.width > 0 && m.width < 68 {
			panelWidth = m.width - 4
		}
		// TrayInput button "bake" has width = len("bake") + 4 = 8.
		// leftW = panelWidth - 8. Inner field is leftW - 4 = panelWidth - 12.
		m.urlInput.SetWidth(panelWidth - 12)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.cancelCtx()
			return m, tea.Quit

		case "ctrl+s":
			if m.stage == StageInput || m.stage == StageBaked {
				m.stage = StageSettings
				m.settingsIndex = 0
				return m, nil
			}

		case "esc":
			if m.stage == StageSettings {
				if m.editingDir {
					m.editingDir = false
					return m, nil
				}
				m.stage = StageInput
				return m, nil
			}
			if m.stage == StageDest {
				if m.isCustomDest {
					m.isCustomDest = false
					return m, nil
				}
				if len(m.portions) > 0 {
					m.stage = StageSelecting
					return m, nil
				}
				m.stage = StageInput
				return m, nil
			}
			if m.stage == StageProbing || m.stage == StageBaking {
				m.cancelCtx()
				m.stage = StageInput
				return m, nil
			}
			if m.stage == StageDirectPrompt || m.stage == StageTorrentPrompt || m.stage == StagePlaylistPrompt || m.stage == StageSelecting {
				m.stage = StageInput
				return m, nil
			}

		case "tab":
			if m.stage == StageInput && m.clipboardURL != "" {
				m.urlInput.SetValue(m.clipboardURL)
				m.historyPos = -1
				return m, nil
			}

		case "d", "D":
			if m.stage == StageDest && !m.isCustomDest {
				opts := BuildDestOptions(m.config.OutDir)
				for _, o := range opts {
					if o.Key == "D" {
						m.chosenDest = o.Path
						m.stage = StageBaking
						m.bakingProgress = engine.BakeProgress{TotalParts: 1}
						return m, m.startBake()
					}
				}
			}

		case "v", "V":
			if m.stage == StageDest && !m.isCustomDest {
				opts := BuildDestOptions(m.config.OutDir)
				for _, o := range opts {
					if o.Key == "V" {
						m.chosenDest = o.Path
						m.stage = StageBaking
						m.bakingProgress = engine.BakeProgress{TotalParts: 1}
						return m, m.startBake()
					}
				}
			}

		case "c", "C":
			if m.stage == StageDest && !m.isCustomDest {
				opts := BuildDestOptions(m.config.OutDir)
				for _, o := range opts {
					if o.Key == "C" {
						m.chosenDest = o.Path
						m.stage = StageBaking
						m.bakingProgress = engine.BakeProgress{TotalParts: 1}
						return m, m.startBake()
					}
				}
			}

		case "o", "O":
			if m.stage == StageDest && !m.isCustomDest {
				m.isCustomDest = true
				m.destInput.SetValue(m.config.OutDir)
				m.destInput.Focus()
				return m, nil
			}

		case "enter":
			switch m.stage {
			case StageSettings:
				item := SettingItems[m.settingsIndex]
				if item.Key == "outDir" {
					if m.editingDir {
						val := strings.TrimSpace(m.settingsDir.Value())
						if val != "" {
							m.config.OutDir = units.ResolveUserPath(val)
							_ = system.SaveConfig(m.config)
						}
						m.editingDir = false
						return m, nil
					}
					m.editingDir = true
					m.settingsDir.SetValue(m.config.OutDir)
					m.settingsDir.Focus()
					return m, nil
				}
				CycleConfig(&m.config, item.Key, 1)
				_ = system.SaveConfig(m.config)
				return m, nil

			case StageInput:
				val := strings.TrimSpace(m.urlInput.Value())
				if val == "" {
					return m, nil
				}
				system.AddToHistory(val)
				m.history = system.LoadHistory()
				m.historyPos = -1
				m.stage = StageProbing
				m.statusText = "probing target…"
				return m, tea.Batch(m.spinner.Tick, m.startInspect(val))

			case StageDirectPrompt:
				if m.promptChoice == 0 {
					if m.config.AskSaveDir && m.initialOutDir == "" {
						m.stage = StageDest
						m.isCustomDest = false
						m.destIndex = 0
						m.destTargetTitle = m.target.Filename
						m.destTargetSub = fmt.Sprintf("%d connections · direct file", m.config.Connections)
						return m, nil
					}
					m.chosenDest = m.config.OutDir
					m.stage = StageBaking
					m.bakingProgress = engine.BakeProgress{TotalParts: 1}
					return m, m.startBake()
				}
				m.stage = StageInput
				return m, nil

			case StageTorrentPrompt:
				if m.promptChoice == 0 {
					if m.config.AskSaveDir && m.initialOutDir == "" {
						m.stage = StageDest
						m.isCustomDest = false
						m.destIndex = 0
						m.destTargetTitle = m.target.Name
						m.destTargetSub = "BitTorrent P2P transfer · aria2c"
						return m, nil
					}
					m.chosenDest = m.config.OutDir
					m.stage = StageBaking
					m.bakingProgress = engine.BakeProgress{TotalParts: 1}
					return m, m.startBake()
				}
				m.stage = StageInput
				return m, nil

			case StagePlaylistPrompt:
				if m.promptChoice == 0 {
					m.isPlaylistMode = true
					plPortions := engine.ExtractPlaylistPortions(&engine.ExtractPortionsOptions{
						EmbedMetadata:  &m.config.EmbedMetadata,
						VideoContainer: m.config.VideoContainer,
						AudioFormat:    m.config.AudioFormat,
					})
					m.portions = plPortions
					m.selectedPortion = 0
					m.stage = StageSelecting
					return m, nil
				}
				// Single track
				m.isPlaylistMode = false
				m.stage = StageProbing
				m.statusText = "probing single track…"
				return m, tea.Batch(m.spinner.Tick, m.startProbeVideo(m.target))

			case StageSelecting:
				if m.target != nil && m.target.Type == core.TargetArchive && m.archivePost != nil {
					if m.selectedPortion == 0 {
						// Download all files
						m.selectedArchiveFiles = m.archivePost.Files
					} else if m.selectedPortion-1 < len(m.archivePost.Files) {
						// Download single file
						m.selectedArchiveFiles = []engine.ArchiveFile{m.archivePost.Files[m.selectedPortion-1]}
					}
					if m.config.AskSaveDir && m.initialOutDir == "" {
						m.stage = StageDest
						m.isCustomDest = false
						m.destIndex = 0
						m.destTargetTitle = m.archivePost.Title
						m.destTargetSub = fmt.Sprintf("Archive batch (%d items)", len(m.selectedArchiveFiles))
						return m, nil
					}
					m.chosenDest = m.config.OutDir
					m.stage = StageBaking
					m.bakingProgress = engine.BakeProgress{TotalParts: len(m.selectedArchiveFiles)}
					return m, m.startBake()
				}

				if m.config.AskSaveDir && m.initialOutDir == "" {
					m.stage = StageDest
					m.isCustomDest = false
					m.destIndex = 0
					if m.probeResult != nil {
						m.destTargetTitle = m.probeResult.Meta.Title
						m.destTargetSub = m.portions[m.selectedPortion].Label
					}
					return m, nil
				}
				m.chosenDest = m.config.OutDir
				m.stage = StageBaking
				m.bakingProgress = engine.BakeProgress{TotalParts: 1}
				return m, m.startBake()

			case StageDest:
				if m.isCustomDest {
					val := strings.TrimSpace(m.destInput.Value())
					if val == "" {
						val = m.config.OutDir
					}
					m.chosenDest = units.ResolveUserPath(val)
					m.stage = StageBaking
					m.bakingProgress = engine.BakeProgress{TotalParts: 1}
					return m, m.startBake()
				}
				opts := BuildDestOptions(m.config.OutDir)
				if m.destIndex >= 0 && m.destIndex < len(opts) {
					opt := opts[m.destIndex]
					if opt.IsCustom {
						m.isCustomDest = true
						m.destInput.SetValue(m.config.OutDir)
						m.destInput.Focus()
						return m, nil
					}
					m.chosenDest = opt.Path
					m.stage = StageBaking
					m.bakingProgress = engine.BakeProgress{TotalParts: 1}
					return m, m.startBake()
				}

			case StageBaked, StageError:
				m.stage = StageInput
				m.urlInput.Reset()
				m.urlInput.Focus()
				return m, nil
			}

		case "up", "k":
			if m.stage == StageInput {
				if len(m.history) > 0 {
					if m.historyPos == -1 {
						m.draftInput = m.urlInput.Value()
						m.historyPos = 0
					} else if m.historyPos < len(m.history)-1 {
						m.historyPos++
					}
					m.urlInput.SetValue(m.history[m.historyPos])
					m.urlInput.CursorEnd()
				}
				return m, nil
			}
			if m.stage == StageDest && !m.isCustomDest {
				if m.destIndex > 0 {
					m.destIndex--
				}
				return m, nil
			}
			if m.stage == StageSettings {
				if !m.editingDir && m.settingsIndex > 0 {
					m.settingsIndex--
				}
				return m, nil
			}
			if m.stage == StageDirectPrompt || m.stage == StageTorrentPrompt || m.stage == StagePlaylistPrompt {
				if m.promptChoice > 0 {
					m.promptChoice--
				}
				return m, nil
			}
			if m.stage == StageSelecting && m.selectedPortion > 0 {
				m.selectedPortion--
				return m, nil
			}

		case "down", "j":
			if m.stage == StageInput {
				if m.historyPos != -1 {
					if m.historyPos > 0 {
						m.historyPos--
						m.urlInput.SetValue(m.history[m.historyPos])
						m.urlInput.CursorEnd()
					} else {
						m.historyPos = -1
						m.urlInput.SetValue(m.draftInput)
						m.urlInput.CursorEnd()
					}
				}
				return m, nil
			}
			if m.stage == StageDest && !m.isCustomDest {
				opts := BuildDestOptions(m.config.OutDir)
				if m.destIndex < len(opts)-1 {
					m.destIndex++
				}
				return m, nil
			}
			if m.stage == StageSettings {
				if !m.editingDir && m.settingsIndex < len(SettingItems)-1 {
					m.settingsIndex++
				}
				return m, nil
			}
			if m.stage == StageDirectPrompt || m.stage == StageTorrentPrompt || m.stage == StagePlaylistPrompt {
				if m.promptChoice < 1 {
					m.promptChoice++
				}
				return m, nil
			}
			if m.stage == StageSelecting && m.selectedPortion < len(m.portions)-1 {
				m.selectedPortion++
				return m, nil
			}

		case "left", "h":
			if m.stage == StageSettings && !m.editingDir {
				item := SettingItems[m.settingsIndex]
				CycleConfig(&m.config, item.Key, -1)
				_ = system.SaveConfig(m.config)
				return m, nil
			}

		case "right", "l", " ":
			if m.stage == StageSettings && !m.editingDir {
				item := SettingItems[m.settingsIndex]
				CycleConfig(&m.config, item.Key, 1)
				_ = system.SaveConfig(m.config)
				return m, nil
			}
		}

	case updateCheckMsg:
		m.latestVersion = msg.latestVersion
		return m, nil

	case bakeProcessingMsg:
		m.processing = true
		return m, nil

	case inspectMsg:
		if msg.err != nil {
			m.stage = StageError
			m.errText = msg.err.Error()
			return m, nil
		}
		m.target = msg.target

		switch msg.target.Type {
		case core.TargetDirect:
			m.promptChoice = 0
			m.stage = StageDirectPrompt
			return m, nil

		case core.TargetTorrent:
			m.promptChoice = 0
			m.stage = StageTorrentPrompt
			return m, nil

		case core.TargetArchive:
			m.archivePost = msg.target.ArchivePost
			system.AddToHistory(msg.target.URL)
			m.history = system.LoadHistory()
			m.historyPos = -1

			if msg.target.ArchivePost != nil && len(msg.target.ArchivePost.Files) > 1 {
				// Multiple files: prompt user to download all or pick a specific file
				var portions []engine.Portion
				portions = append(portions, engine.Portion{
					Label: fmt.Sprintf("📦 all files (%d items) · %s", len(msg.target.ArchivePost.Files), msg.target.ArchivePost.Title),
					Kind:  engine.PortionKindVideo,
				})
				for _, f := range msg.target.ArchivePost.Files {
					portions = append(portions, engine.Portion{
						Label: fmt.Sprintf("📄 %s", f.Name),
						Kind:  engine.PortionKindVideo,
					})
				}
				m.portions = portions
				m.selectedPortion = 0
				m.stage = StageSelecting
				return m, nil
			}

			// Single file
			if msg.target.ArchivePost != nil && len(msg.target.ArchivePost.Files) == 1 {
				m.selectedArchiveFiles = msg.target.ArchivePost.Files
			}
			if m.config.AskSaveDir && m.initialOutDir == "" {
				m.stage = StageDest
				m.isCustomDest = false
				m.destIndex = 0
				if msg.target.ArchivePost != nil {
					m.destTargetTitle = msg.target.ArchivePost.Title
				}
				m.destTargetSub = "Archive file download"
				return m, nil
			}
			m.chosenDest = m.config.OutDir
			m.stage = StageBaking
			m.bakingProgress = engine.BakeProgress{TotalParts: 1}
			return m, m.startBake()

		case core.TargetVideo:
			m.statusText = "extracting formats with yt-dlp…"
			return m, m.startProbeVideo(msg.target)
		}

	case probePlaylistMsg:
		if msg.err != nil {
			m.stage = StageError
			m.errText = msg.err.Error()
			return m, nil
		}
		m.playlistMeta = msg.meta
		m.promptChoice = 0
		m.stage = StagePlaylistPrompt
		return m, nil

	case probeResultMsg:
		if msg.err != nil {
			m.stage = StageError
			m.errText = msg.err.Error()
			return m, nil
		}
		m.probeResult = msg.probeResult
		m.portions = msg.portions
		m.selectedPortion = 0

		// Auto-selection based on preferQuality
		if len(m.portions) > 0 {
			chosenIdx := -1
			if m.config.PreferQuality == "best" {
				chosenIdx = 0
			} else if m.config.PreferQuality == "audio" {
				for i, p := range m.portions {
					if p.Kind == engine.PortionKindAudio {
						chosenIdx = i
						break
					}
				}
			} else if m.config.PreferQuality == "1080p" {
				for i, p := range m.portions {
					if strings.HasPrefix(p.Label, "1080p") {
						chosenIdx = i
						break
					}
				}
			}

			if chosenIdx != -1 {
				m.selectedPortion = chosenIdx
				if m.config.AskSaveDir && m.initialOutDir == "" {
					m.stage = StageDest
					m.isCustomDest = false
					m.destIndex = 0
					m.destTargetTitle = m.probeResult.Meta.Title
					m.destTargetSub = m.portions[chosenIdx].Label
					return m, nil
				}
				m.chosenDest = m.config.OutDir
				m.stage = StageBaking
				m.bakingProgress = engine.BakeProgress{TotalParts: 1}
				return m, m.startBake()
			}
		}

		m.stage = StageSelecting
		return m, nil

	case bakeProgressMsg:
		m.bakingProgress = engine.BakeProgress(msg)
		return m, nil

	case bakeDoneMsg:
		if msg.err != nil {
			m.stage = StageError
			m.errText = msg.err.Error()
			return m, nil
		}
		m.stage = StageBaked
		m.resultFilePath = msg.path
		m.FinalPath = msg.path
		return m, nil

	case spinner.TickMsg:
		if m.stage == StageProbing || m.stage == StageBaking {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	var cmd tea.Cmd
	if m.stage == StageInput {
		prevVal := m.urlInput.Value()
		m.urlInput, cmd = m.urlInput.Update(msg)
		if m.urlInput.Value() != prevVal {
			m.historyPos = -1
		}
		return m, cmd
	} else if m.stage == StageDest && m.isCustomDest {
		m.destInput, cmd = m.destInput.Update(msg)
		return m, cmd
	} else if m.stage == StageSettings && m.editingDir {
		m.settingsDir, cmd = m.settingsDir.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() tea.View {
	panelWidth := 64
	if m.width > 0 && m.width < 68 {
		panelWidth = m.width - 4
	}

	mascot := RenderMascot(panelWidth)
	tagline := lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(styleRegular.Render("feed a link, bake a file."))
	hintsLine := lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(styleDim.Render("youtube · x · instagram · soundcloud · torrent · and more"))

	var headerBlock string
	if m.latestVersion != "" && (m.stage == StageInput || m.stage == StageBaked) {
		updBanner := lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(
			styleTitle.Render("✦ update available: ") +
				styleDim.Render(m.version) +
				styleTitle.Render(" → ") +
				styleSuccess.Render("v"+m.latestVersion) +
				styleDim.Render(" (run: curl -fsSL https://raw.githubusercontent.com/KabosuNeko/anpan/main/install.sh | bash)"),
		)
		headerBlock = fmt.Sprintf("%s\n\n%s\n%s\n\n%s", mascot, tagline, hintsLine, updBanner)
	} else {
		headerBlock = fmt.Sprintf("%s\n\n%s\n%s", mascot, tagline, hintsLine)
	}

	var stageBlock string

	switch m.stage {
	case StageInput:
		m.urlInput.SetWidth(panelWidth - 12)
		tray := RenderTrayInput("url / magnet / file", panelWidth, m.urlInput.View(), "bake", strings.TrimSpace(m.urlInput.Value()) == "")
		stageBlock = tray

	case StageProbing:
		spin := m.spinner.View() + " " + styleDim.Render(m.statusText)
		stageBlock = lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(spin)

	case StageDirectPrompt:
		header := styleRegular.Render(units.Truncate(m.target.Filename, panelWidth))
		sub := styleDim.Render(fmt.Sprintf("%d connections · direct file", m.config.Connections))
		info := lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(fmt.Sprintf("%s\n%s", header, sub))

		choices := []string{
			fmt.Sprintf("download file (%d connections)", m.config.Connections),
			"cancel",
		}
		var choiceLines []string
		for i, c := range choices {
			if i == m.promptChoice {
				choiceLines = append(choiceLines, styleDim.Render("❯ ")+styleRegular.Render(c))
			} else {
				choiceLines = append(choiceLines, "  "+styleDim.Render(c))
			}
		}
		card := RenderBunCard("direct file download", panelWidth, strings.Join(choiceLines, "\n"))
		stageBlock = fmt.Sprintf("%s\n\n%s", info, card)

	case StageTorrentPrompt:
		header := styleRegular.Render(units.Truncate(m.target.Name, panelWidth))
		sub := styleDim.Render("BitTorrent P2P transfer · aria2c")
		info := lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(fmt.Sprintf("%s\n%s", header, sub))

		choices := []string{
			"start BitTorrent download (P2P)",
			"cancel",
		}
		var choiceLines []string
		for i, c := range choices {
			if i == m.promptChoice {
				choiceLines = append(choiceLines, styleDim.Render("❯ ")+styleRegular.Render(c))
			} else {
				choiceLines = append(choiceLines, "  "+styleDim.Render(c))
			}
		}
		card := RenderBunCard("bittorrent transfer", panelWidth, strings.Join(choiceLines, "\n"))
		stageBlock = fmt.Sprintf("%s\n\n%s", info, card)

	case StagePlaylistPrompt:
		title := m.playlistMeta.Title
		header := styleRegular.Render(units.Truncate(title, panelWidth))
		sub := styleDim.Render(fmt.Sprintf("%s · %d tracks", m.playlistMeta.Uploader, m.playlistMeta.TrackCount))
		info := lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(fmt.Sprintf("%s\n%s", header, sub))

		choices := []string{
			fmt.Sprintf("download full playlist (%d tracks)", m.playlistMeta.TrackCount),
			"download single track only",
		}
		var choiceLines []string
		for i, c := range choices {
			if i == m.promptChoice {
				choiceLines = append(choiceLines, styleDim.Render("❯ ")+styleRegular.Render(c))
			} else {
				choiceLines = append(choiceLines, "  "+styleDim.Render(c))
			}
		}
		card := RenderBunCard("playlist detected", panelWidth, strings.Join(choiceLines, "\n"))
		stageBlock = fmt.Sprintf("%s\n\n%s", info, card)

	case StageSelecting:
		title := "Media"
		sub := ""
		if m.probeResult != nil {
			title = m.probeResult.Meta.Title
			sub = m.probeResult.Meta.Uploader
			if m.probeResult.Meta.Duration != nil {
				sub += " · " + units.FormatDuration(*m.probeResult.Meta.Duration)
			}
		} else if m.target != nil && m.target.Type == core.TargetArchive && m.archivePost != nil {
			title = m.archivePost.Title
			sub = fmt.Sprintf("%d files · %s", len(m.archivePost.Files), m.archivePost.Service)
		}
		header := styleRegular.Render(units.Truncate(title, panelWidth))
		info := lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(fmt.Sprintf("%s\n%s", header, styleDim.Render(sub)))

		var items []string
		for i, p := range m.portions {
			if i == m.selectedPortion {
				items = append(items, styleDim.Render("❯ ")+styleRegular.Render(p.Label))
			} else {
				items = append(items, "  "+styleDim.Render(p.Label))
			}
		}
		card := RenderBunCard("format", panelWidth, strings.Join(items, "\n"))
		stageBlock = fmt.Sprintf("%s\n\n%s", info, card)

	case StageDest:
		m.destInput.SetWidth(panelWidth - 14)
		stageBlock = RenderDestView(panelWidth, m.destTargetTitle, m.destTargetSub, m.isCustomDest, m.destIndex, m.destInput, m.config.OutDir)

	case StageBaking:
		title := "Download"
		if m.destTargetTitle != "" {
			title = m.destTargetTitle
		} else if m.target != nil && m.target.Name != "" {
			title = m.target.Name
		} else if m.probeResult != nil && m.probeResult.Meta.Title != "" {
			title = m.probeResult.Meta.Title
		}
		header := lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(styleRegular.Render(units.Truncate(title, panelWidth)))

		var content string
		if m.processing {
			spin := m.spinner.View() + " " + styleDim.Render("processing / merging…")
			content = lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(spin)
		} else if m.bakingProgress.DownloadedBytes > 0 || m.bakingProgress.TotalBytes != nil {
			var partTag string
			if m.bakingProgress.PlaylistItem > 0 && m.bakingProgress.PlaylistTotal > 0 {
				partTag = fmt.Sprintf("[%d/%d] ", m.bakingProgress.PlaylistItem, m.bakingProgress.PlaylistTotal)
			} else if m.bakingProgress.TotalParts > 1 {
				partTag = fmt.Sprintf("[%d/%d] ", m.bakingProgress.Part, m.bakingProgress.TotalParts)
			}

			var extraConn string
			if m.bakingProgress.Seeders != nil {
				extraConn = fmt.Sprintf("  · P2P (%d peers, %d seeds)", m.bakingProgress.Connections, *m.bakingProgress.Seeders)
			} else if m.config.Aria2c {
				extraConn = fmt.Sprintf("  · aria2c (%d)", m.config.Connections)
			}

			if m.bakingProgress.TotalBytes != nil && *m.bakingProgress.TotalBytes > 0 {
				pct := m.bakingProgress.DownloadedBytes / *m.bakingProgress.TotalBytes
				bar := RenderCrustBar(pct, min(40, panelWidth-10))
				barCentered := lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(bar)

				speed := ""
				if m.bakingProgress.Speed > 0 {
					speed = units.FormatSpeed(m.bakingProgress.Speed)
				}
				eta := ""
				if m.bakingProgress.ETA > 0 {
					eta = units.FormatEta(m.bakingProgress.ETA) + " left"
				}
				metaLine := fmt.Sprintf("%s%10s  %-12s%s", partTag, speed, eta, extraConn)
				stats := lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(styleDim.Render(strings.TrimSpace(metaLine)))
				content = fmt.Sprintf("%s\n%s", barCentered, stats)
			} else {
				bytesStr := units.FormatBytes(m.bakingProgress.DownloadedBytes)
				speedStr := ""
				if m.bakingProgress.Speed > 0 {
					speedStr = units.FormatSpeed(m.bakingProgress.Speed)
				}
				spinPrefix := ""
				if m.bakingProgress.DownloadedBytes == 0 {
					spinPrefix = m.spinner.View() + " "
				}
				metaLine := fmt.Sprintf("%s%s%8s  %-10s%s", spinPrefix, partTag, bytesStr, speedStr, extraConn)
				content = lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(styleDim.Render(strings.TrimSpace(metaLine)))
			}
		} else {
			extra := ""
			if m.config.Aria2c {
				extra = fmt.Sprintf("  · aria2c (%d)", m.config.Connections)
			}
			conn := m.spinner.View() + " " + styleDim.Render("connecting to server…"+extra)
			content = lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(conn)
		}

		stageBlock = fmt.Sprintf("%s\n\n%s", header, content)

	case StageBaked:
		status := lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(styleRegular.Render("downloaded"))
		home, _ := os.UserHomeDir()
		path := lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(styleDim.Render(units.ShortenPath(m.resultFilePath, home, panelWidth)))
		again := lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(styleRegular.Render("↵ download another"))
		stageBlock = fmt.Sprintf("%s\n%s\n\n%s", status, path, again)

	case StageError:
		errHeader := lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(styleError.Render("download failed"))
		errLines := units.WrapText(m.errText, panelWidth)
		var wrapped []string
		for _, l := range errLines {
			wrapped = append(wrapped, lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(styleDim.Render(l)))
		}
		retry := lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(styleRegular.Render("↵ retry"))
		stageBlock = fmt.Sprintf("%s\n%s\n\n%s", errHeader, strings.Join(wrapped, "\n"), retry)

	case StageSettings:
		m.settingsDir.SetWidth(22)
		stageBlock = RenderSettingsView(panelWidth, m.settingsIndex, m.editingDir, m.settingsDir, m.config)
	}

	hints := stageHints[m.stage]
	footer := lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(RenderFooterHints(hints))

	boxContent := fmt.Sprintf("%s\n\n%s\n\n%s", headerBlock, stageBlock, footer)

	// Center horizontally and vertically inside terminal viewport
	vWidth := m.width
	if vWidth <= 0 {
		vWidth = 80
	}
	vHeight := m.height
	if vHeight <= 0 {
		vHeight = 24
	}

	placed := lipgloss.Place(vWidth, vHeight, lipgloss.Center, lipgloss.Center, boxContent)

	v := tea.NewView(placed)
	v.AltScreen = true
	return v
}
