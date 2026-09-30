package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// tabLabels is indexed by tabID, so the bar and the keybindings cannot drift
// apart as tabs are added
var tabLabels = [TabCount]string{
	TabPattern:       "Pattern",
	TabInfo:          "Info",
	TabVisualization: "Visualization",
}

var tabIcons = [TabCount]NerdIcon{
	TabPattern:       IconMusic,
	TabInfo:          IconInfo,
	TabVisualization: IconSineWave,
}

func (m Model) tabBar() string {
	inactive := lipgloss.NewStyle().Padding(0, 1)

	// the active tab inverts: a dark foreground on the accent, so it reads as
	// selected without needing a second border to track
	active := lipgloss.NewStyle().
		Padding(0, 1).
		Background(m.Theme.Accent).
		Foreground(m.Theme.StatusBg)

	var bar strings.Builder

	for tab := range TabCount {
		label := string(tabIcons[tab]) + "  " + tabLabels[tab]

		if tab == m.ActiveTab {
			bar.WriteString(active.Render(label))
			continue
		}

		bar.WriteString(inactive.Render(label))
	}

	return bar.String()
}

func (m Model) tabBarLine() string {
	return lipgloss.NewStyle().
		Width(m.Width).
		Height(1).
		MaxWidth(m.Width).
		MaxHeight(1).
		Render(m.tabBar())
}
