package tui

import (
	"fmt"
	"strconv"
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

	case "seek":
		order, row, ok := ParseSeek(fields[1:])
		if !ok {
			m.Status = "usage: :seek <order> [row]"
			return m, nil
		}

		cmd := m.SeekTo(order, row)
		return m, cmd

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

func ParseSeek(args []string) (order, row int, ok bool) {
	if len(args) < 1 || len(args) > 2 {
		return 0, 0, false
	}

	var err error
	if order, err = strconv.Atoi(args[0]); err != nil {
		return 0, 0, false
	}

	if len(args) == 2 {
		if row, err = strconv.Atoi(args[1]); err != nil {
			return 0, 0, false
		}
	}

	return order, row, true
}

func nextTab(m *Model) tea.Cmd {
	m.setTab(tabID((uint8(m.ActiveTab) + 1) % uint8(TabCount)))
	return nil
}

func prevTab(m *Model) tea.Cmd {
	m.setTab(tabID((uint8(m.ActiveTab) + uint8(TabCount) - 1) % uint8(TabCount)))
	return nil
}

func toggleFollow(m *Model) tea.Cmd {
	tab := &m.Tabs[TabPattern]
	tab.Follow = !tab.Follow

	if !tab.Follow {
		return nil
	}

	m.FollowPatternRow(m.Position.Row)

	if m.Position.Order < 0 || m.Position.Order == m.LoadedOrder {
		return nil
	}

	m.LoadedOrder = m.Position.Order
	return loadPattern(m.Module, m.Position.Pattern)
}

// SeekTo moves the module to an order and row, and points the tab
// at whatever pattern that order holds
func (m *Model) SeekTo(order, row int) tea.Cmd {
	switch {
	case order < 0 || order >= m.Module.NumOrders():
		m.Status = fmt.Sprintf("order %d is outside the module", order)
		return nil

	case !m.Module.IsOrderPlayable(order):
		m.Status = fmt.Sprintf("order %d is a skip or stop marker", order)
		return nil
	}

	if err := m.Player.Seek(order, row); err != nil {
		m.Status = err.Error()
		return nil
	}

	m.Position = m.Module.Position()

	tab := &m.Tabs[TabPattern]
	if order == m.LoadedOrder {
		tab.Pattern.Clamp(m.Info.Channels)
		return nil
	}

	m.LoadedOrder = order
	tab.Pattern.Row = row
	tab.Pattern.Clamp(m.Info.Channels)

	return loadPattern(m.Module, m.Position.Pattern)
}

// AdjacentOrder returns the orders around a given one, wrapping
// around the sequence and stepping over skip and stop entries
func (m *Model) AdjacentOrder(order, step int) int {
	total := m.Module.NumOrders()
	if total <= 0 {
		return -1
	}

	// nothing loaded yet, so start walking from the beginning
	order = max(order, 0)

	for range total {
		order = (order + step + total) % total
		if m.Module.IsOrderPlayable(order) {
			return order
		}
	}

	return -1
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
	var load tea.Cmd
	if !m.Tabs[TabPattern].Follow && !m.Playing {
		load = m.SeekTo(m.LoadedOrder, m.Tabs[TabPattern].Pattern.Row)
	}

	if err := m.Player.Resume(); err != nil {
		m.Status = err.Error()
		return load
	}

	m.Playing = true
	m.refreshViewport(TabInfo)

	return tea.Batch(load, rearmPlayback(m.Audio, m.Player))
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
