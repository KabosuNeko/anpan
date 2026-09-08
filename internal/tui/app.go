package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

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
	StageSearch         StageName = "search"
	StageSeedPrompt     StageName = "seed_prompt"
	StageSeeding        StageName = "seeding"
)

var stageHints = map[StageName][][2]string{
	StageInput: {
		{"↑↓", "history"},
		{"↵", "bake"},
		{"^f", "search torrent"},
		{"^e", "seed"},
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
		{"tab/[/]", "tab"},
		{"↑↓", "select"},
		{"⇄", "cycle"},
		{"↵", "edit"},
		{"esc", "close"},
	},
	StageSearch: {
		{"↑↓←→", "move"},
		{"↵/d", "download"},
		{"o", "folder"},
		{"y", "copy"},
		{"s", "sort"},
		{"/", "search"},
		{"tab", "switch"},
		{"esc", "back"},
		{"^c", "quit"},
	},
	StageSeedPrompt: {
		{"↵", "seed"},
		{"esc", "cancel"},
		{"^c", "quit"},
	},
	StageSeeding: {
		{"y", "copy magnet"},
		{"esc", "stop"},
		{"^c", "quit"},
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

type searchResultsMsg struct {
	results []engine.TorrentSearchResult
	err     error
}

type seedCreatedMsg struct {
	torrentPath string
	magnet      string
	infoHash    string
	name        string
	err         error
}

type toastClearMsg struct{}

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
	version       string
	latestVersion string
	initialURL    string
	initialOutDir string
	width         int
	height        int
	stage         StageName
	statusText    string
	errText       string
	config        system.AnpanConfig

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
	settingsTab   int
	settingsIndex int
	editingDir    bool
	settingsDir   textinput.Model

	// Browse & Search stage
	browseCategory  string
	browseRegion    string
	browseInput     textinput.Model
	browseResults   []engine.TorrentSearchResult
	browseCursor    int
	browsePage      int
	browseSort      string
	browseSearching bool
	browseStatus     string
	browseDetail     *engine.TorrentSearchResult
	browseMinSeeds   int
	searchHistory    []string
	searchHistoryPos int
	draftSearch      string

	// Seeding stage
	seedInput       textinput.Model
	seedingActive   bool
	seedingName     string
	seedingMagnet   string
	seedingProgress engine.BakeProgress
	seedingCancel   context.CancelFunc

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

	bi := textinput.New()
	bi.Prompt = ""
	bi.Placeholder = "Search or paste a magnet link…"
	bi.CharLimit = 256

	si := textinput.New()
	si.Prompt = ""
	si.Placeholder = "/path/to/folder or file to seed..."
	si.CharLimit = 512

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styleRegular

	ctx, cancel := context.WithCancel(context.Background())

	cfg := system.LoadConfig()
	if cfg.ColorTheme != "" {
		ApplyTheme(cfg.ColorTheme)
	}

	defaultSort := "seeds"
	if cfg.DefaultSearchSort != "" {
		defaultSort = cfg.DefaultSearchSort
	}
	defaultCat := "all"
	if cfg.DefaultSearchCat != "" {
		defaultCat = cfg.DefaultSearchCat
	}

	m := Model{
		version:          version,
		initialURL:       initialURL,
		initialOutDir:    initialOutDir,
		stage:            StageInput,
		config:           cfg,
		bridge:           &ProgramBridge{},
		history:          system.LoadHistory(),
		historyPos:       -1,
		searchHistoryPos: -1,
		urlInput:         ti,
		destInput:        di,
		settingsDir:      sdi,
		browseInput:      bi,
		seedInput:        si,
		browseCategory:   defaultCat,
		browseRegion:     "content",
		browseSort:       defaultSort,
		browsePage:       1,
		browseMinSeeds:   0,
		spinner:          s,
		width:            80,
		height:           24,
		ctx:              ctx,
		cancelCtx:        cancel,
	}

	clip := system.ReadClipboard()
	if clip != "" && core.IsLikelyTarget(clip) {
		m.clipboardURL = clip
		if cfg.AutoPaste && initialURL == "" {
			m.urlInput.SetValue(clip)
		} else {
			m.urlInput.Placeholder = clip + "  ⇥ paste"
		}
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
			VideoCodec:     m.config.VideoCodec,
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
				SpeedLimit:  m.config.SpeedLimit,
			}, handlers)
			return bakeDoneMsg{path: path, err: err}
		}

		if m.target.Type == core.TargetTorrent {
			aria2c, err := engine.FindAria2c()
			if err != nil {
				return bakeDoneMsg{err: fmt.Errorf("aria2c required for torrents: %w", err)}
			}
			path, err := engine.BakeTorrentDownload(m.ctx, engine.TorrentDownloadOptions{
				Aria2cBin:  aria2c,
				Target:     m.target.Target,
				OutputDir:  outDir,
				SpeedLimit: m.config.SpeedLimit,
				SeedRatio:  m.config.TorrentSeedRatio,
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
					Headers:  f.Headers,
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
				SpeedLimit:  m.config.SpeedLimit,
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
			SpeedLimit:     m.config.SpeedLimit,
		}, handlers)
		return bakeDoneMsg{path: path, err: err}
	}
}

func (m Model) startSearch(query, category string, page int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 12*time.Second)
		defer cancel()
		if page < 1 {
			page = 1
		}
		results, err := engine.SearchTorrents(ctx, query, &engine.SearchOptions{
			Category: category,
			Limit:    100,
			SortBy:   m.browseSort,
			Page:     page,
		})
		return searchResultsMsg{results: results, err: err}
	}
}

func sortSearchResults(results []engine.TorrentSearchResult, sortBy string) {
	switch strings.ToLower(sortBy) {
	case "seeds":
		sort.SliceStable(results, func(i, j int) bool {
			if results[i].Seeders == results[j].Seeders {
				return results[i].SizeBytes > results[j].SizeBytes
			}
			return results[i].Seeders > results[j].Seeders
		})
	case "size", "size-desc":
		sort.SliceStable(results, func(i, j int) bool {
			return results[i].SizeBytes > results[j].SizeBytes
		})
	case "size-asc", "size-up", "smallest":
		sort.SliceStable(results, func(i, j int) bool {
			if results[i].SizeBytes <= 0 && results[j].SizeBytes > 0 {
				return false
			}
			if results[i].SizeBytes > 0 && results[j].SizeBytes <= 0 {
				return true
			}
			if results[i].SizeBytes == results[j].SizeBytes {
				return results[i].Seeders > results[j].Seeders
			}
			return results[i].SizeBytes < results[j].SizeBytes
		})
	case "peers", "leechers", "activity":
		sort.SliceStable(results, func(i, j int) bool {
			pi := results[i].Seeders + results[i].Leechers
			pj := results[j].Seeders + results[j].Leechers
			if pi == pj {
				return results[i].Seeders > results[j].Seeders
			}
			return pi > pj
		})
	case "source":
		sort.SliceStable(results, func(i, j int) bool {
			si := strings.ToLower(results[i].Source)
			sj := strings.ToLower(results[j].Source)
			if si == sj {
				return results[i].Seeders > results[j].Seeders
			}
			return si < sj
		})
	case "name":
		sort.SliceStable(results, func(i, j int) bool {
			return strings.ToLower(results[i].Title) < strings.ToLower(results[j].Title)
		})
	}
}

func (m Model) getFilteredBrowseResults() []engine.TorrentSearchResult {
	if m.browseMinSeeds <= 0 {
		return m.browseResults
	}
	var filtered []engine.TorrentSearchResult
	for _, r := range m.browseResults {
		if r.Seeders >= m.browseMinSeeds {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func (m Model) startSeed(targetPath string) tea.Cmd {
	return func() tea.Msg {
		res, err := engine.CreateTorrent(targetPath, "", nil)
		if err != nil {
			return seedCreatedMsg{err: err}
		}
		_ = system.WriteClipboard(res.Magnet)
		name := filepath.Base(targetPath)
		return seedCreatedMsg{
			torrentPath: res.TorrentPath,
			magnet:      res.Magnet,
			infoHash:    res.InfoHash,
			name:        name,
		}
	}
}

func (m *Model) startSeedingProcess(torrentPath, dataDir string) tea.Cmd {
	seedCtx, cancel := context.WithCancel(context.Background())
	m.seedingCancel = cancel

	return func() tea.Msg {
		aria2c, err := engine.FindAria2c()
		if err != nil {
			return errMsg(err)
		}
		handlers := engine.BakeHandlers{
			OnProgress: func(p engine.BakeProgress) {
				m.bridge.Send(bakeProgressMsg(p))
			},
		}
		_, _ = engine.BakeTorrentSeed(seedCtx, engine.TorrentSeedOptions{
			Aria2cBin:   aria2c,
			TorrentPath: torrentPath,
			OutputDir:   dataDir,
			UploadLimit: m.config.SpeedLimit,
		}, handlers)
		return nil
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		panelWidth := 70
		if m.width > 0 && m.width < 74 {
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
			if m.seedingCancel != nil {
				m.seedingCancel()
			}
			return m, tea.Quit

		case "ctrl+s":
			if m.stage == StageInput || m.stage == StageBaked {
				m.stage = StageSettings
				m.settingsTab = 0
				m.settingsIndex = 0
				return m, nil
			}

		case "ctrl+f":
			m.stage = StageSearch
			m.browseCategory = "all"
			if m.config.DefaultSearchCat != "" {
				m.browseCategory = m.config.DefaultSearchCat
			}
			m.browseRegion = "search"
			m.browseInput.Focus()
			m.browsePage = 1
			if len(m.browseResults) == 0 {
				m.browseSearching = true
				return m, tea.Batch(m.spinner.Tick, m.startSearch("", m.browseCategory, 1))
			}
			return m, nil

		case "ctrl+e":
			m.stage = StageSeedPrompt
			m.seedInput.Reset()
			m.seedInput.Focus()
			return m, nil

		case "esc":
			if m.stage == StageSearch {
				if m.browseRegion == "detail" {
					m.browseRegion = "content"
					m.browseDetail = nil
					return m, nil
				}
				if m.browseRegion == "search" {
					m.browseRegion = "content"
					m.browseInput.Blur()
					return m, nil
				}
				m.stage = StageInput
				return m, nil
			}
			if m.stage == StageSeedPrompt {
				m.stage = StageInput
				return m, nil
			}
			if m.stage == StageSeeding {
				if m.seedingCancel != nil {
					m.seedingCancel()
				}
				m.seedingActive = false
				m.stage = StageInput
				return m, nil
			}
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
			if m.stage == StageSettings && !m.editingDir {
				m.settingsTab = (m.settingsTab + 1) % len(SettingCategories)
				m.settingsIndex = 0
				return m, nil
			}
			if m.stage == StageSearch {
				switch m.browseRegion {
				case "sidebar":
					m.browseRegion = "search"
					m.browseInput.Focus()
				case "search":
					m.browseRegion = "content"
					m.browseInput.Blur()
				case "content":
					m.browseRegion = "sidebar"
				case "detail":
					m.browseRegion = "content"
					m.browseDetail = nil
				default:
					m.browseRegion = "sidebar"
				}
				return m, nil
			}
			if m.stage == StageInput && m.clipboardURL != "" {
				m.urlInput.SetValue(m.clipboardURL)
				m.historyPos = -1
				return m, nil
			}

		case "shift+tab":
			if m.stage == StageSettings && !m.editingDir {
				m.settingsTab = (m.settingsTab - 1 + len(SettingCategories)) % len(SettingCategories)
				m.settingsIndex = 0
				return m, nil
			}
			if m.stage == StageSearch {
				switch m.browseRegion {
				case "sidebar":
					m.browseRegion = "content"
				case "search":
					m.browseRegion = "sidebar"
					m.browseInput.Blur()
				case "content":
					m.browseRegion = "search"
					m.browseInput.Focus()
				case "detail":
					m.browseRegion = "content"
					m.browseDetail = nil
				default:
					m.browseRegion = "sidebar"
				}
				return m, nil
			}

		case "d":
			if m.stage == StageSearch {
				results := m.getFilteredBrowseResults()
				if (m.browseRegion == "content" && len(results) > m.browseCursor) || (m.browseRegion == "detail" && m.browseDetail != nil) {
					var r engine.TorrentSearchResult
					if m.browseRegion == "detail" && m.browseDetail != nil {
						r = *m.browseDetail
					} else {
						r = results[m.browseCursor]
					}
					m.target = &core.TargetInspection{
						Type:   core.TargetTorrent,
						Target: r.Magnet,
						Name:   r.Title,
					}
					if m.config.AskSaveDir && m.initialOutDir == "" {
						m.stage = StageDest
						m.isCustomDest = false
						m.destIndex = 0
						m.destTargetTitle = r.Title
						m.destTargetSub = fmt.Sprintf("BitTorrent · %s", r.Source)
						return m, nil
					}
					m.chosenDest = m.config.OutDir
					m.stage = StageBaking
					m.bakingProgress = engine.BakeProgress{TotalParts: 1}
					return m, m.startBake()
				}
			}
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

		case "D":
			if m.stage == StageSearch {
				results := m.getFilteredBrowseResults()
				if (m.browseRegion == "content" && len(results) > m.browseCursor) || (m.browseRegion == "detail" && m.browseDetail != nil) {
					var r engine.TorrentSearchResult
					if m.browseRegion == "detail" && m.browseDetail != nil {
						r = *m.browseDetail
					} else {
						r = results[m.browseCursor]
					}
					m.target = &core.TargetInspection{
						Type:   core.TargetTorrent,
						Target: r.Magnet,
						Name:   r.Title,
					}
					m.stage = StageDest
					m.isCustomDest = false
					m.destIndex = 0
					m.destTargetTitle = r.Title
					m.destTargetSub = fmt.Sprintf("BitTorrent · %s", r.Source)
					return m, nil
				}
			}
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

		case "i", "I":
			if m.stage == StageSearch {
				if m.browseRegion == "detail" {
					m.browseRegion = "content"
					m.browseDetail = nil
					return m, nil
				}
				results := m.getFilteredBrowseResults()
				if (m.browseRegion == "content" || m.browseRegion == "sidebar") && len(results) > m.browseCursor {
					m.browseRegion = "detail"
					selected := results[m.browseCursor]
					m.browseDetail = &selected
					return m, nil
				}
			}

		case "f", "F":
			if m.stage == StageSearch && m.browseRegion != "search" {
				switch m.browseMinSeeds {
				case 0:
					m.browseMinSeeds = 1
				case 1:
					m.browseMinSeeds = 5
				case 5:
					m.browseMinSeeds = 20
				case 20:
					m.browseMinSeeds = 0
				default:
					m.browseMinSeeds = 0
				}
				results := m.getFilteredBrowseResults()
				if m.browseCursor >= len(results) {
					if len(results) > 0 {
						m.browseCursor = len(results) - 1
					} else {
						m.browseCursor = 0
					}
				}
				if m.browseMinSeeds > 0 {
					m.browseStatus = fmt.Sprintf("Filter: Seeders ≥ %d (%d items)", m.browseMinSeeds, len(results))
				} else {
					m.browseStatus = fmt.Sprintf("Filter: All seeds (%d items)", len(results))
				}
				return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
					return toastClearMsg{}
				})
			}

		case "o", "O":
			if m.stage == StageSearch {
				results := m.getFilteredBrowseResults()
				if (m.browseRegion == "content" && len(results) > m.browseCursor) || (m.browseRegion == "detail" && m.browseDetail != nil) {
					var r engine.TorrentSearchResult
					if m.browseRegion == "detail" && m.browseDetail != nil {
						r = *m.browseDetail
					} else {
						r = results[m.browseCursor]
					}
					m.target = &core.TargetInspection{
						Type:   core.TargetTorrent,
						Target: r.Magnet,
						Name:   r.Title,
					}
					m.stage = StageDest
					m.isCustomDest = false
					m.destIndex = 0
					m.destTargetTitle = r.Title
					m.destTargetSub = fmt.Sprintf("BitTorrent · %s", r.Source)
					return m, nil
				}
			}
			if m.stage == StageDest && !m.isCustomDest {
				m.isCustomDest = true
				m.destInput.SetValue(m.config.OutDir)
				m.destInput.Focus()
				return m, nil
			}

		case "y", "Y":
			if m.stage == StageSearch {
				results := m.getFilteredBrowseResults()
				if m.browseRegion == "content" && len(results) > m.browseCursor {
					r := results[m.browseCursor]
					_ = system.WriteClipboard(r.Magnet)
					m.browseStatus = "Magnet copied to clipboard!"
					return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
						return toastClearMsg{}
					})
				}
				if m.browseRegion == "detail" && m.browseDetail != nil {
					_ = system.WriteClipboard(m.browseDetail.Magnet)
					m.browseStatus = "Magnet copied to clipboard!"
					return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
						return toastClearMsg{}
					})
				}
			}
			if m.stage == StageSeeding && m.seedingMagnet != "" {
				_ = system.WriteClipboard(m.seedingMagnet)
				return m, nil
			}

		case "s":
			if m.stage == StageSearch && m.browseRegion != "search" {
				switch m.browseSort {
				case "seeds":
					m.browseSort = "size"
				case "size":
					m.browseSort = "size-asc"
				case "size-asc":
					m.browseSort = "peers"
				case "peers":
					m.browseSort = "name"
				case "name":
					m.browseSort = "source"
				case "source":
					m.browseSort = "seeds"
				default:
					m.browseSort = "seeds"
				}
				sortSearchResults(m.browseResults, m.browseSort)
				m.browseStatus = "Sorted by " + FormatSortLabel(m.browseSort)
				return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
					return toastClearMsg{}
				})
			}

		case "[", "<", "p":
			if m.stage == StageSettings && !m.editingDir {
				m.settingsTab = (m.settingsTab - 1 + len(SettingCategories)) % len(SettingCategories)
				m.settingsIndex = 0
				return m, nil
			}
			if m.stage == StageSearch && m.browseRegion != "search" {
				if m.browsePage > 1 {
					m.browsePage--
					m.browseSearching = true
					m.browseCursor = 0
					return m, tea.Batch(m.spinner.Tick, m.startSearch(m.browseInput.Value(), m.browseCategory, m.browsePage))
				}
				return m, nil
			}

		case "]", ">", "n":
			if m.stage == StageSettings && !m.editingDir {
				m.settingsTab = (m.settingsTab + 1) % len(SettingCategories)
				m.settingsIndex = 0
				return m, nil
			}
			if m.stage == StageSearch && m.browseRegion != "search" {
				m.browsePage++
				m.browseSearching = true
				m.browseCursor = 0
				return m, tea.Batch(m.spinner.Tick, m.startSearch(m.browseInput.Value(), m.browseCategory, m.browsePage))
			}

		case "1", "2", "3", "4":
			if m.stage == StageSettings && !m.editingDir {
				tIdx := int(msg.String()[0] - '1')
				if tIdx >= 0 && tIdx < len(SettingCategories) {
					m.settingsTab = tIdx
					m.settingsIndex = 0
					return m, nil
				}
			}

		case "pgdown", "ctrl+d":
			if m.stage == StageSearch && m.browseRegion == "content" {
				results := m.getFilteredBrowseResults()
				m.browseCursor += 10
				if m.browseCursor >= len(results) {
					m.browseCursor = len(results) - 1
				}
				if m.browseCursor < 0 {
					m.browseCursor = 0
				}
				return m, nil
			}

		case "pgup", "ctrl+u":
			if m.stage == StageSearch && m.browseRegion == "content" {
				m.browseCursor -= 10
				if m.browseCursor < 0 {
					m.browseCursor = 0
				}
				return m, nil
			}

		case "home":
			if m.stage == StageSearch && m.browseRegion == "content" {
				m.browseCursor = 0
				return m, nil
			}

		case "end":
			if m.stage == StageSearch && m.browseRegion == "content" {
				results := m.getFilteredBrowseResults()
				if len(results) > 0 {
					m.browseCursor = len(results) - 1
				}
				return m, nil
			}

		case "/":
			if m.stage == StageSearch && m.browseRegion != "search" {
				m.browseRegion = "search"
				m.browseInput.Focus()
				return m, nil
			}

		case "enter":
			switch m.stage {
			case StageSearch:
				if m.browseRegion == "search" {
					q := strings.TrimSpace(m.browseInput.Value())
					if q != "" {
						if len(m.searchHistory) == 0 || m.searchHistory[0] != q {
							m.searchHistory = append([]string{q}, m.searchHistory...)
						}
					}
					m.searchHistoryPos = -1
					m.draftSearch = ""
					m.browseSearching = true
					m.browseRegion = "content"
					m.browseInput.Blur()
					m.browsePage = 1
					return m, tea.Batch(m.spinner.Tick, m.startSearch(q, m.browseCategory, 1))
				}
				if m.browseRegion == "sidebar" {
					m.browseRegion = "content"
					return m, nil
				}
				results := m.getFilteredBrowseResults()
				if (m.browseRegion == "content" && len(results) > m.browseCursor) || (m.browseRegion == "detail" && m.browseDetail != nil) {
					var r engine.TorrentSearchResult
					if m.browseRegion == "detail" && m.browseDetail != nil {
						r = *m.browseDetail
					} else {
						r = results[m.browseCursor]
					}
					m.target = &core.TargetInspection{
						Type:   core.TargetTorrent,
						Target: r.Magnet,
						Name:   r.Title,
					}
					if m.config.AskSaveDir && m.initialOutDir == "" {
						m.stage = StageDest
						m.isCustomDest = false
						m.destIndex = 0
						m.destTargetTitle = r.Title
						m.destTargetSub = fmt.Sprintf("BitTorrent · %s", r.Source)
						return m, nil
					}
					m.chosenDest = m.config.OutDir
					m.stage = StageBaking
					m.bakingProgress = engine.BakeProgress{TotalParts: 1}
					return m, m.startBake()
				}
				return m, nil

			case StageSeedPrompt:
				val := strings.TrimSpace(m.seedInput.Value())
				if val == "" {
					return m, nil
				}
				clean := core.CleanLocalPath(val)
				if _, err := os.Stat(clean); err != nil {
					m.stage = StageError
					m.errText = fmt.Sprintf("Seed target not found: %s", clean)
					return m, nil
				}
				m.stage = StageProbing
				m.statusText = "creating .torrent and seeding…"
				return m, tea.Batch(m.spinner.Tick, m.startSeed(clean))

			case StageSettings:
				if m.editingDir {
					val := strings.TrimSpace(m.settingsDir.Value())
					if val != "" {
						m.config.OutDir = units.ResolveUserPath(val)
						_ = system.SaveConfig(m.config)
					}
					m.editingDir = false
					return m, nil
				}
				if m.settingsTab < 0 || m.settingsTab >= len(SettingCategories) {
					m.settingsTab = 0
				}
				cat := SettingCategories[m.settingsTab]
				if m.settingsIndex >= 0 && m.settingsIndex < len(cat.Items) {
					item := cat.Items[m.settingsIndex]
					if item.Key == "outDir" {
						m.editingDir = true
						m.settingsDir.SetValue(m.config.OutDir)
						m.settingsDir.Focus()
						return m, nil
					}
					CycleConfig(&m.config, item.Key, 1)
					_ = system.SaveConfig(m.config)
				}
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
						VideoCodec:     m.config.VideoCodec,
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
			if m.stage == StageSearch {
				if m.browseRegion == "search" {
					if len(m.searchHistory) > 0 {
						if m.searchHistoryPos == -1 {
							m.draftSearch = m.browseInput.Value()
							m.searchHistoryPos = 0
						} else if m.searchHistoryPos < len(m.searchHistory)-1 {
							m.searchHistoryPos++
						}
						m.browseInput.SetValue(m.searchHistory[m.searchHistoryPos])
						m.browseInput.CursorEnd()
					}
					return m, nil
				}
				if m.browseRegion == "sidebar" {
					idx := 0
					for i, c := range BrowseCategories {
						if c.Key == m.browseCategory {
							idx = i
							break
						}
					}
					if idx > 0 {
						idx--
					} else {
						idx = len(BrowseCategories) - 1
					}
					m.browseCategory = BrowseCategories[idx].Key
					if m.browseCategory != "seeding" {
						m.browseSearching = true
						m.browseCursor = 0
						m.browsePage = 1
						return m, tea.Batch(m.spinner.Tick, m.startSearch(m.browseInput.Value(), m.browseCategory, 1))
					}
					return m, nil
				}
				if m.browseRegion == "content" {
					if m.browseCursor > 0 {
						m.browseCursor--
					} else if m.browseCursor == 0 {
						m.browseRegion = "search"
						m.browseInput.Focus()
					}
					return m, nil
				}
			}
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
			if m.stage == StageSearch {
				if m.browseRegion == "search" {
					if m.searchHistoryPos != -1 {
						if m.searchHistoryPos > 0 {
							m.searchHistoryPos--
							m.browseInput.SetValue(m.searchHistory[m.searchHistoryPos])
							m.browseInput.CursorEnd()
						} else {
							m.searchHistoryPos = -1
							m.browseInput.SetValue(m.draftSearch)
							m.browseInput.CursorEnd()
						}
					}
					return m, nil
				}
				if m.browseRegion == "sidebar" {
					idx := 0
					for i, c := range BrowseCategories {
						if c.Key == m.browseCategory {
							idx = i
							break
						}
					}
					if idx < len(BrowseCategories)-1 {
						idx++
					} else {
						idx = 0
					}
					m.browseCategory = BrowseCategories[idx].Key
					if m.browseCategory != "seeding" {
						m.browseSearching = true
						m.browseCursor = 0
						m.browsePage = 1
						return m, tea.Batch(m.spinner.Tick, m.startSearch(m.browseInput.Value(), m.browseCategory, 1))
					}
					return m, nil
				}
				if m.browseRegion == "content" {
					results := m.getFilteredBrowseResults()
					if m.browseCursor < len(results)-1 {
						m.browseCursor++
					}
					return m, nil
				}
			}
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
				if m.settingsTab < 0 || m.settingsTab >= len(SettingCategories) {
					m.settingsTab = 0
				}
				cat := SettingCategories[m.settingsTab]
				if !m.editingDir && m.settingsIndex < len(cat.Items)-1 {
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
			if m.stage == StageSearch && m.browseRegion != "search" {
				if m.browseRegion == "content" {
					m.browseRegion = "sidebar"
					return m, nil
				}
			}
			if m.stage == StageSettings && !m.editingDir {
				if m.settingsTab < 0 || m.settingsTab >= len(SettingCategories) {
					m.settingsTab = 0
				}
				cat := SettingCategories[m.settingsTab]
				if m.settingsIndex >= 0 && m.settingsIndex < len(cat.Items) {
					item := cat.Items[m.settingsIndex]
					CycleConfig(&m.config, item.Key, -1)
					_ = system.SaveConfig(m.config)
				}
				return m, nil
			}

		case "right", "l", " ", "space":
			if m.stage == StageSearch && m.browseRegion != "search" {
				if m.browseRegion == "sidebar" {
					m.browseRegion = "content"
					return m, nil
				}
				if m.browseRegion == "content" {
					results := m.getFilteredBrowseResults()
					if len(results) > m.browseCursor {
						m.browseRegion = "detail"
						selected := results[m.browseCursor]
						m.browseDetail = &selected
						return m, nil
					}
				}
				if m.browseRegion == "detail" {
					m.browseRegion = "content"
					m.browseDetail = nil
					return m, nil
				}
			}
			if m.stage == StageSettings && !m.editingDir {
				if m.settingsTab < 0 || m.settingsTab >= len(SettingCategories) {
					m.settingsTab = 0
				}
				cat := SettingCategories[m.settingsTab]
				if m.settingsIndex >= 0 && m.settingsIndex < len(cat.Items) {
					item := cat.Items[m.settingsIndex]
					CycleConfig(&m.config, item.Key, 1)
					_ = system.SaveConfig(m.config)
				}
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

		case core.TargetSearch:
			m.stage = StageSearch
			m.browseCategory = "all"
			m.browseRegion = "content"
			m.browseInput.SetValue(msg.target.SearchQuery)
			m.browseInput.Blur()
			m.browseSearching = true
			m.browseCursor = 0
			m.browsePage = 1
			return m, tea.Batch(m.spinner.Tick, m.startSearch(msg.target.SearchQuery, "all", 1))

		case core.TargetSeed:
			m.stage = StageSeedPrompt
			m.seedInput.SetValue(msg.target.Target)
			m.seedInput.Focus()
			return m, nil
		}

	case searchResultsMsg:
		m.browseSearching = false
		if msg.err != nil {
			if m.browsePage > 1 {
				m.browsePage--
				m.browseStatus = fmt.Sprintf("Page error: %s (returned to page %d)", msg.err.Error(), m.browsePage)
			} else {
				m.browseStatus = "Search failed: " + msg.err.Error()
			}
			return m, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
				return toastClearMsg{}
			})
		}
		if len(msg.results) == 0 && m.browsePage > 1 {
			failedPage := m.browsePage
			m.browsePage--
			m.browseStatus = fmt.Sprintf("No more results on page %d (returned to page %d)", failedPage, m.browsePage)
			return m, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
				return toastClearMsg{}
			})
		}
		m.browseResults = msg.results
		sortSearchResults(m.browseResults, m.browseSort)
		m.browseCursor = 0
		if m.browsePage > 1 {
			m.browseStatus = fmt.Sprintf("Loaded page %d (%d items)", m.browsePage, len(m.browseResults))
			return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
				return toastClearMsg{}
			})
		}
		return m, nil
	case seedCreatedMsg:
		if msg.err != nil {
			m.stage = StageError
			m.errText = msg.err.Error()
			return m, nil
		}
		m.seedingActive = true
		m.seedingName = msg.name
		m.seedingMagnet = msg.magnet
		m.stage = StageSeeding
		if m.config.Notifications {
			system.SendNotification("anpan", fmt.Sprintf("✓ Seeding started: %s", msg.name))
		}
		return m, m.startSeedingProcess(msg.torrentPath, filepath.Dir(msg.torrentPath))

	case toastClearMsg:
		m.browseStatus = ""
		return m, nil

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
			if m.config.Notifications {
				system.SendNotification("anpan", fmt.Sprintf("✗ Download failed: %s", msg.err.Error()))
			}
			return m, nil
		}
		m.stage = StageBaked
		m.resultFilePath = msg.path
		m.FinalPath = msg.path

		// Fetch and save synced lyrics for audio tracks
		if m.config.Lyrics != "off" && len(m.portions) > m.selectedPortion && m.portions[m.selectedPortion].Kind == engine.PortionKindAudio {
			trackTitle := ""
			artist := ""
			duration := float64(0)
			if m.probeResult != nil {
				trackTitle = m.probeResult.Meta.Title
				artist = m.probeResult.Meta.Uploader
				if m.probeResult.Meta.Duration != nil {
					duration = *m.probeResult.Meta.Duration
				}
			}
			if trackTitle != "" {
				go func(audioPath, title, art string, dur float64) {
					if lyr, err := engine.FetchLyrics(context.Background(), title, art, dur); err == nil && lyr != nil {
						_, _ = engine.SaveLrcFile(audioPath, lyr)
					}
				}(msg.path, trackTitle, artist, duration)
			}
		}

		if m.config.Notifications {
			itemTitle := filepath.Base(msg.path)
			if m.probeResult != nil && m.probeResult.Meta.Title != "" {
				itemTitle = m.probeResult.Meta.Title
			} else if m.playlistMeta != nil && m.playlistMeta.Title != "" {
				itemTitle = m.playlistMeta.Title
			} else if m.archivePost != nil && m.archivePost.Title != "" {
				itemTitle = m.archivePost.Title
			} else if m.target != nil && m.target.Name != "" {
				itemTitle = m.target.Name
			}
			system.SendNotification("anpan", fmt.Sprintf("✓ Finished: %s", itemTitle))
		}
		return m, nil

	case spinner.TickMsg:
		if m.stage == StageProbing || m.stage == StageBaking || (m.stage == StageSearch && m.browseSearching) {
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
	} else if m.stage == StageSearch && m.browseRegion == "search" {
		prevVal := m.browseInput.Value()
		m.browseInput, cmd = m.browseInput.Update(msg)
		if m.browseInput.Value() != prevVal {
			m.searchHistoryPos = -1
		}
		return m, cmd
	} else if m.stage == StageSeedPrompt {
		m.seedInput, cmd = m.seedInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() tea.View {
	vWidth := m.width
	if vWidth <= 0 {
		vWidth = 80
	}
	vHeight := m.height
	if vHeight <= 0 {
		vHeight = 24
	}

	// PRO MAX SEARCH UI:
	// Clean, dedicated, maximized viewport without ASCII mascot or header clutter
	if m.stage == StageSearch {
		searchWidth := vWidth - 2
		if searchWidth > vWidth {
			searchWidth = vWidth
		}
		if searchWidth < 48 {
			searchWidth = 48
		}
		browseState := BrowseViewState{
			Width:           searchWidth,
			Height:          vHeight,
			ActiveCategory:  m.browseCategory,
			FocusedRegion:   m.browseRegion,
			SearchInput:     m.browseInput.Value(),
			Results:         m.getFilteredBrowseResults(),
			Cursor:          m.browseCursor,
			Page:            m.browsePage,
			SortMode:        m.browseSort,
			Searching:       m.browseSearching,
			StatusMsg:       m.browseStatus,
			DetailResult:    m.browseDetail,
			MinSeeds:        m.browseMinSeeds,
			SeedingActive:   m.seedingActive,
			SeedingName:     m.seedingName,
			SeedingProgress: m.bakingProgress,
			SeedingMagnet:   m.seedingMagnet,
		}
		searchView := RenderBrowseView(browseState)
		placed := lipgloss.Place(vWidth, vHeight, lipgloss.Center, lipgloss.Center, searchView)
		v := tea.NewView(placed)
		v.AltScreen = true
		return v
	}

	panelWidth := 70
	if m.width > 0 && m.width < 74 {
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
				styleDim.Render(" (run: anpan update)"),
		)
		headerBlock = fmt.Sprintf("%s\n\n%s\n%s\n\n%s", mascot, tagline, hintsLine, updBanner)
	} else {
		headerBlock = fmt.Sprintf("%s\n\n%s\n%s", mascot, tagline, hintsLine)
	}

	var stageBlock string

	switch m.stage {
	case StageSeedPrompt:
		m.seedInput.SetWidth(panelWidth - 8)
		box := RenderTrayInput("seed file or directory", panelWidth, m.seedInput.View(), "seed", strings.TrimSpace(m.seedInput.Value()) == "")
		info := lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(styleDim.Render("Enter local file or folder path. A .torrent and magnet link will be generated."))
		stageBlock = fmt.Sprintf("%s\n\n%s", info, box)

	case StageSeeding:
		title := m.seedingName
		if title == "" {
			title = "Seeding Active"
		}
		header := lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(styleRegular.Render(units.Truncate(title, panelWidth)))
		speedStr := ""
		if m.bakingProgress.Speed > 0 {
			speedStr = units.FormatSpeed(m.bakingProgress.Speed) + " UL"
		}
		bytesStr := units.FormatBytes(m.bakingProgress.DownloadedBytes) + " uploaded"
		peersStr := fmt.Sprintf("%d peers", m.bakingProgress.Connections)
		statsLine := lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(styleSuccess.Render(fmt.Sprintf("%s  ·  %s  ·  %s", speedStr, bytesStr, peersStr)))
		magLine := lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(styleDim.Render("Magnet copied to clipboard! (press y to re-copy)"))
		stageBlock = fmt.Sprintf("%s\n\n%s\n\n%s", header, statsLine, magLine)

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
		warning := styleDim.Render("⚠ P2P swarm exposes your public IP. Consider using a VPN.")
		info := lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(fmt.Sprintf("%s\n%s\n\n%s", header, sub, warning))

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
		stageBlock = RenderSettingsView(panelWidth, m.settingsTab, m.settingsIndex, m.editingDir, m.settingsDir, m.config)
	}

	hints := stageHints[m.stage]
	if m.stage == StageInput && panelWidth < 70 {
		hints = [][2]string{
			{"↵", "bake"},
			{"^f", "search torrent"},
			{"^e", "seed"},
			{"^s", "settings"},
			{"^c", "quit"},
		}
	}
	if m.stage == StageInput && panelWidth < 58 {
		hints = [][2]string{
			{"↵", "bake"},
			{"^f", "search torrent"},
			{"^c", "quit"},
		}
	}
	footer := lipgloss.NewStyle().Width(panelWidth).Align(lipgloss.Center).Render(RenderFooterHints(hints))

	boxContent := fmt.Sprintf("%s\n\n%s\n\n%s", headerBlock, stageBlock, footer)

	placed := lipgloss.Place(vWidth, vHeight, lipgloss.Center, lipgloss.Center, boxContent)

	v := tea.NewView(placed)
	v.AltScreen = true
	return v
}
