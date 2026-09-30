package tui

import (
	"time"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/Lakelimbo/nezumi/libopenmpt"
)

type tabID uint8

const (
	TabPattern tabID = iota
	TabInfo
	TabVisualization
	TabCount
)

type visualizationID uint8

const (
	visualizationWaveform visualizationID = iota
	visualizationSpectrum
	visualizationBars
)

type TabState struct {
	Visualizer visualizationID
	Viewport   viewport.Model

	// Pattern is the pattern tab's scroll state: which slice of the grid is
	// on screen. It carries the row selection, both scroll offsets and the
	// body size, because the grid is drawn from it on every frame
	Pattern PatternView

	// Follow keeps the row the player is on visible and selected
	Follow bool

	// Formatted is the loaded pattern, decoded once. Unstyled, so the theme
	// can change without touching it
	Formatted [][]CellFields
}

type ModuleInfo struct {
	Title    string
	Artist   string
	Tracker  string
	TypeLong string
	Duration time.Duration
	Channels int
	Patterns int
}

type audioState struct {
	Wave []float32
	RMS  float64
}

type Model struct {
	Module *libopenmpt.Module
	Player *libopenmpt.Player
	Audio  <-chan libopenmpt.PCMFrame

	// TO-DO:
	// Settings will be here eventually

	ActiveTab tabID
	Tabs      [TabCount]TabState

	Width  int
	Height int

	Position    libopenmpt.Position
	LoadedOrder int
	Pattern     libopenmpt.PatternData
	Info        ModuleInfo
	Playing     bool
	Status      string

	CommandActive bool
	Command       textinput.Model

	PendingKeys []string
	LeaderToken uint64

	AudioState audioState

	Theme Theme
}

type positionMsg struct {
	Position libopenmpt.Position
}

type PatternMsg struct {
	Index int
	Data  libopenmpt.PatternData
	Err   error
}

type audioMsg struct {
	Frame libopenmpt.PCMFrame
}

type audioStoppedMsg struct{}
type playbackDoneMsg struct{}

type leaderExpiredMsg struct {
	Token uint64
}

func NewTabs() [TabCount]TabState {
	tabs := [TabCount]TabState{}

	tabs[TabPattern].Follow = true

	return tabs
}

func New(
	mod *libopenmpt.Module,
	player *libopenmpt.Player,
	audio <-chan libopenmpt.PCMFrame,
) Model {
	theme := DefaultTheme()

	command := textinput.New()
	command.Placeholder = "command"

	return Model{
		Module:      mod,
		Player:      player,
		Audio:       audio,
		ActiveTab:   TabPattern,
		LoadedOrder: -1,
		Playing:     true,
		Command:     command,
		Info: ModuleInfo{
			Title:    mod.Title(),
			Artist:   mod.Artist(),
			Tracker:  mod.Tracker(),
			TypeLong: mod.TypeLong(),
			Duration: mod.Duration(),
			Channels: mod.NumChannels(),
			Patterns: mod.NumPatterns(),
		},
		AudioState: audioState{
			Wave: make([]float32, maxWaveSamples),
		},
		Tabs:  NewTabs(),
		Theme: theme,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		pollPosition(m.Module),
		rearmPlayback(m.Audio, m.Player),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.resize()

		return m, nil

	case positionMsg:
		m.Position = msg.Position
		m.refreshViewport(TabInfo)

		return m, m.FollowPosition()

	case PatternMsg:
		switch {
		case msg.Err != nil:
			m.Status = msg.Err.Error()
			m.LoadedOrder = -1

		case msg.Index == m.Module.OrderPattern(m.LoadedOrder):
			m.LoadPattern(msg.Data)
			m.FollowPatternRow(m.Position.Row)
		}

		return m, pollPosition(m.Module)

	case audioMsg:
		m.ingestAudio(msg.Frame)

		if m.ActiveTab == TabVisualization {
			m.refreshViewport(TabVisualization)
		}

		return m, waitForAudio(m.Audio, m.Player.Done())

	case audioStoppedMsg:
		if m.Playing {
			return m, waitForAudio(m.Audio, m.Player.Done())
		}

		return m, nil

	case playbackDoneMsg:
		if m.Playing {
			return m, nil
		}

		m.Playing = false
		m.refreshViewport(TabInfo)

		return m, nil

	case leaderExpiredMsg:
		// ignore a timeout belonging to a sequence that has already been
		// completed or replaced
		if msg.Token == m.LeaderToken {
			m.PendingKeys = nil
		}

		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// FollowPosition take the position's poll. When following, the band owns
// the cursor, the window, and the pattern, following into the next order.
func (m *Model) FollowPosition() tea.Cmd {
	if m.Tabs[TabPattern].Follow {
		m.FollowPatternRow(m.Position.Row)

		if order := m.Position.Order; order >= 0 && order != m.LoadedOrder {
			m.LoadedOrder = order

			return loadPattern(m.Module, m.Position.Pattern)
		}
	}

	return pollPosition(m.Module)
}

func (m *Model) bodySize() (width, height int) {
	width = max(0, m.Width)
	height = max(0, m.Height-chromeHeight)

	return width, height
}

func (m *Model) resize() {
	w, h := m.bodySize()

	for t := range m.Tabs {
		m.Tabs[t].Viewport.SetWidth(w)
		m.Tabs[t].Viewport.SetHeight(h)
	}

	pattern := &m.Tabs[TabPattern].Pattern
	pattern.Width = w
	pattern.Height = h

	// the offsets are absolute positions, so a resize does not invalidate
	// them, but it can push the scroll window past the new bounds. The
	// selection never moves: the column that is highlighted is a property of
	// the pattern, not of how much of it happens to be on screen
	pattern.Clamp(m.Info.Channels)

	m.refreshAllViewports()
}
