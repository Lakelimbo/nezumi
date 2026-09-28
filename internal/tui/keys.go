package tui

import (
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

const (
	LeaderKey  = "\\"
	leaderWait = 700 * time.Millisecond
)

var leaderCommands = map[string]func(*Model) tea.Cmd{
	"t": nextTab,
	"T": prevTab,
	"f": toggleFollow,
	" ": togglePlayback,
	"q": func(*Model) tea.Cmd { return tea.Quit },
}

func leaderKeyList() string {
	keys := make([]string, 0, len(leaderCommands))
	for key := range leaderCommands {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	return strings.Join(keys, " ")
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.CommandActive {
		return m.handleCommandKey(msg)
	}

	// a half-typed leader sequence swallows the next key, so `\t` is a
	// command rather than a literal tab switch.
	if len(m.PendingKeys) != 0 {
		return m.handleLeaderKey(msg)
	}

	key := msg.String()

	if key == LeaderKey {
		m.LeaderToken++
		m.PendingKeys = []string{LeaderKey}

		return m, leaderTimeout(m.LeaderToken)
	}

	switch key {
	case "tab":
		return m.runLeaderCommand("t")

	case "shift+tab":
		return m.runLeaderCommand("T")

	case "1":
		m.setTab(TabPattern)
		return m, nil

	case "2":
		m.setTab(TabInfo)
		return m, nil

	case "3":
		m.setTab(TabVisualization)
		return m, nil

	case "space":
		return m.runLeaderCommand(" ")

	case ":":
		return m.openCommandLine()

	case "q", "ctrl+c":
		return m, tea.Quit
	}

	if m.ActiveTab == TabPattern {
		return m.handlePatternKey(msg)
	}

	return m.forwardToViewport(msg)
}

func (m Model) runLeaderCommand(key string) (tea.Model, tea.Cmd) {
	run, ok := leaderCommands[key]
	if !ok {
		return m, nil
	}

	// run against the local copy and only then return it.
	//
	// Returning `m, run(&m)` would depend on the interface
	// conversion of m happening after the call, which is not
	// something to rely on
	cmd := run(&m)

	return m, cmd
}

func (m Model) handleLeaderKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	m.PendingKeys = nil

	// repeating the leader starts a new sequence instead of doubling up
	if key == LeaderKey {
		m.LeaderToken++
		m.PendingKeys = []string{LeaderKey}

		return m, leaderTimeout(m.LeaderToken)
	}

	run, ok := leaderCommands[key]
	if !ok {
		m.Status = "no command for " + LeaderKey + key
		return m, nil
	}

	cmd := run(&m)

	return m, cmd
}

func (m Model) openCommandLine() (tea.Model, tea.Cmd) {
	m.CommandActive = true
	m.Command.SetValue("")
	// the prompt takes a column of the line, so the input gets the rest
	m.Command.SetWidth(max(1, m.Width-1))

	return m, m.Command.Focus()
}

func (m Model) handleCommandKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.closeCommandLine()
		return m, nil

	case "enter":
		draft := m.Command.Value()
		m.closeCommandLine()

		return executeCommand(m, draft)
	}

	var cmd tea.Cmd
	m.Command, cmd = m.Command.Update(msg)

	return m, cmd
}

func (m *Model) closeCommandLine() {
	m.CommandActive = false
	m.Command.Blur()
	m.Command.Reset()
}

func (m Model) handlePatternKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	tab := &m.Tabs[TabPattern]
	view, ch := &tab.Pattern, m.Info.Channels
	page := view.ViewRows()

	switch msg.String() {
	case "up", "k":
		view.MoveRow(-1, ch)
	case "down", "j":
		view.MoveRow(1, ch)
	case "pgup", "b":
		view.MoveRow(-page, ch)
	case "pgdown", "f":
		view.MoveRow(page, ch)
	case "g", "home":
		view.GotoTop(ch)
	case "G", "end":
		view.GotoBottom(ch)
	case "right", "l":
		view.ScrollCols(1, ch)
	case "left", "h":
		view.ScrollCols(-1, ch)
	case "L", "shift+right":
		view.ScrollCols(ChannelColumnWidth, ch)
	case "H", "shift+left":
		view.ScrollCols(-ChannelColumnWidth, ch)

	case "F":
		m.Status = ""
		tab.Follow = !tab.Follow
		m.FollowPatternRow(m.Position.Row)
	}

	return m, nil
}

func (m Model) forwardToViewport(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	viewport := m.Tabs[m.ActiveTab].Viewport
	viewport, cmd = viewport.Update(msg)
	m.Tabs[m.ActiveTab].Viewport = viewport

	return m, cmd
}
