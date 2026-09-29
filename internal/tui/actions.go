package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

func executeCommand(m Model, raw string) (tea.Model, tea.Cmd) {
	fields := strings.Fields(strings.ToLower(raw))
	if len(fields) == 0 {
		return m, nil
	}

	m.Status = ""

	switch fields[0] {
	case "tab":
		tab, ok := parseTab(fields[1:])
		if !ok {
			m.Status = "usage: :tab pattern|info|visualization"
			return m, nil
		}

		m.setTab(tab)

	case "view":
		view, ok := parseVisualization(fields[1:])
		if !ok {
			m.Status = "usage: :view waveform|spectrum|bars"
			return m, nil
		}

		m.Tabs[TabVisualization].Visualizer = view
		m.refreshViewport(TabVisualization)

	case "play", "resume":
		cmd := m.play()
		return m, cmd

	case "pause":
		m.pause()
		return m, nil

	case "stop":
		m.stop()
		return m, nil

	case "playpause":
		return m, togglePlayback(&m)

	case "follow":
		return m, toggleFollow(&m)

	case "q", "quit":
		return m, tea.Quit

	default:
		m.Status = "unknown command: " + fields[0]
	}

	return m, nil
}

func parseTab(args []string) (tabID, bool) {
	if len(args) != 1 {
		return 0, false
	}

	switch args[0] {
	case "pattern":
		return TabPattern, true
	case "info":
		return TabInfo, true
	case "visualization":
		return TabVisualization, true
	}

	return 0, false
}

func parseVisualization(args []string) (visualizationID, bool) {
	if len(args) != 1 {
		return 0, false
	}

	switch args[0] {
	case "waveform":
		return visualizationWaveform, true
	case "spectrum":
		return visualizationSpectrum, true
	case "bars":
		return visualizationBars, true
	}

	return 0, false
}

func nextTab(m *Model) tea.Cmd {
	m.setTab(tabID((int(m.ActiveTab) + 1) % int(TabCount)))
	return nil
}

func prevTab(m *Model) tea.Cmd {
	m.setTab(tabID((int(m.ActiveTab) + int(TabCount) - 1) % int(TabCount)))
	return nil
}

func toggleFollow(m *Model) tea.Cmd {
	m.Tabs[TabPattern].Follow = !m.Tabs[TabPattern].Follow

	if m.Tabs[TabPattern].Follow {
		m.FollowPatternRow(m.Position.Row)
	} else {
		m.refreshViewport(TabPattern)
	}

	return nil
}

func togglePlayback(m *Model) tea.Cmd {
	if m.Playing {
		m.pause()
		return nil
	}

	return m.play()
}

func stopPlayback(m *Model) tea.Cmd {
	m.stop()

	return nil
}

func (m *Model) play() tea.Cmd {
	if err := m.Player.Resume(); err != nil {
		m.Status = err.Error()
		return nil
	}

	m.Playing = true
	m.refreshViewport(TabInfo)

	return rearmPlayback(m.Audio, m.Player)
}

func (m *Model) pause() {
	m.Player.Pause()
	m.Playing = false
	m.refreshViewport(TabInfo)
}

func (m *Model) stop() {
	if err := m.Player.Rewind(); err != nil {
		m.Status = err.Error()
	}

	m.Playing = false
	m.refreshViewport(TabInfo)
}

func (m *Model) setTab(tab tabID) {
	m.ActiveTab = tab
	m.refreshViewport(tab)
}
