package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

const chromeHeight = 2

func (m Model) View() tea.View {
	// before the first WindowSizeMsg there is nothing to lay out, and a
	// zero-width render would make every component panic or emit garbage
	if m.Width == 0 || m.Height < chromeHeight+1 {
		return tea.NewView("")
	}

	view := tea.NewView(strings.Join([]string{
		m.tabBarLine(),
		m.bodyView(),
		m.FooterLine(),
	}, "\n"))

	view.AltScreen = true
	view.WindowTitle = "nezumi"

	return view
}

func (m Model) bodyView() string {
	w, h := m.bodySize()
	if w == 0 || h == 0 {
		return ""
	}

	if m.ActiveTab == TabPattern {
		return m.PatternGrid()
	}

	return m.Tabs[m.ActiveTab].Viewport.View()
}

func (m *Model) refreshViewport(tab tabID) {
	//nolint:exhaustive
	switch tab {
	case TabPattern:
		m.Tabs[tab].Pattern.Clamp(m.Info.Channels)

	case TabInfo:
		m.Tabs[tab].Viewport.SetContentLines(m.infoContent())

	case TabVisualization:
		m.Tabs[tab].Viewport.SetContentLines(m.visualizationContent())
	}
}

func (m *Model) refreshAllViewports() {
	for tab := range TabCount {
		m.refreshViewport(tab)
	}
}
