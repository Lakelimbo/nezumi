package tui_test

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/Lakelimbo/nezumi/internal/tui"
	"github.com/Lakelimbo/nezumi/internal/utils"
)

// TestViewFillsTerminal is the layout contract: the tab bar and the footer each
// take exactly one row, so the whole screen is used and nothing is pushed off
// the bottom
func TestViewFillsTerminal(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {120, 30}, {200, 50}} {
		width, height := size[0], size[1]

		m := utils.NewTestModel(width, height)
		utils.LoadTestPattern(&m, 64)

		content := m.View().Content

		if got := lipgloss.Height(content); got != height {
			t.Errorf("%dx%d: view is %d rows, want %d", width, height, got, height)
		}

		if got := lipgloss.Width(content); got > width {
			t.Errorf("%dx%d: view is %d columns, want at most %d", width, height, got, width)
		}
	}
}

// TestFooterIsOneRow checks every footer variant, since they all share the one
// row at the bottom and a variant that is taller steals a row from the body
func TestFooterIsOneRow(t *testing.T) {
	m := utils.NewTestModel(100, 30)
	utils.LoadTestPattern(&m, 64)

	footers := map[string]func() string{
		"status":    m.StatusBar,
		"command":   m.CommandLine,
		"which-key": m.WhichKeyLine,
	}

	for name, render := range footers {
		line := render()

		if got := lipgloss.Height(line); got != tui.FooterHeight {
			t.Errorf("%s footer is %d rows, want %d", name, got, tui.FooterHeight)
		}

		if got := lipgloss.Width(line); got != m.Width {
			t.Errorf("%s footer is %d columns, want %d", name, got, m.Width)
		}
	}
}

func TestViewBeforeFirstResize(t *testing.T) {
	m := utils.NewTestModel(0, 0)

	if got := m.View().Content; got != "" {
		t.Errorf("view before the first window size = %q, want empty", got)
	}
}

func TestWhichKeyLineListsLeaderCommands(t *testing.T) {
	m := utils.NewTestModel(100, 30)
	m.PendingKeys = []string{tui.LeaderKey}

	line := m.WhichKeyLine()

	for _, key := range []string{" ", "f", "q", "t", "T"} {
		if !strings.Contains(line, key) {
			t.Errorf("which-key prompt does not offer %q: %q", key, line)
		}
	}
}
