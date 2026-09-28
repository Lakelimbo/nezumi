package tui_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Lakelimbo/nezumi/internal/tui"
	"github.com/Lakelimbo/nezumi/internal/utils"
)

// press feeds one key through the real Update path, so the leader bookkeeping is
// exercised the way the terminal would drive it
func press(t *testing.T, m tui.Model, keys ...string) tui.Model {
	t.Helper()

	for _, key := range keys {
		msg, ok := keyPress(key)
		if !ok {
			t.Fatalf("unrecognized key %q", key)
		}

		updated, _ := m.Update(msg)

		next, ok := updated.(tui.Model)
		if !ok {
			t.Fatalf("Update returned %T, want tui.Model", updated)
		}

		m = next
	}

	return m
}

func TestLeaderSequenceSwitchesTabs(t *testing.T) {
	m := utils.NewTestModel(100, 30)

	if m.ActiveTab != tui.TabPattern {
		t.Fatalf("start on tab %v, want pattern", m.ActiveTab)
	}

	m = press(t, m, tui.LeaderKey)
	if m.ActiveTab != tui.TabPattern {
		t.Fatalf("leader key alone switched to tab %v", m.ActiveTab)
	}

	if len(m.PendingKeys) != 1 {
		t.Fatalf("PendingKeys = %v, want the leader key", m.PendingKeys)
	}

	m = press(t, m, "t")
	if m.ActiveTab != tui.TabInfo {
		t.Errorf("after %st the tab is %v, want info", tui.LeaderKey, m.ActiveTab)
	}

	if m.PendingKeys != nil {
		t.Errorf("PendingKeys = %v after the sequence completed, want nil", m.PendingKeys)
	}
}

func TestLeaderTogglesFollow(t *testing.T) {
	m := utils.NewTestModel(100, 30)
	utils.LoadTestPattern(&m, 64)

	if !m.Tabs[tui.TabPattern].Follow {
		t.Fatal("follow should start on")
	}

	m = press(t, m, tui.LeaderKey, "f")
	if m.Tabs[tui.TabPattern].Follow {
		t.Error("follow still on after toggling it off")
	}

	m = press(t, m, tui.LeaderKey, "f")
	if !m.Tabs[tui.TabPattern].Follow {
		t.Error("follow still off after toggling it back on")
	}
}

func TestLeaderRejectsUnknownCommand(t *testing.T) {
	m := utils.NewTestModel(100, 30)

	m = press(t, m, tui.LeaderKey, "Z")

	if m.Status == "" {
		t.Error("no status message for an unknown leader command")
	}

	if m.PendingKeys != nil {
		t.Errorf("PendingKeys = %v, want the sequence dropped", m.PendingKeys)
	}
}

// TestHorizontalKeys tests the keys to scroll horizontally properly
//
// l or right 			-> scroll right by 1 character
// h or left 				-> scroll left by 1 character
// L or shift+right -> scroll one channel to the right
// H or shift+left 	-> scroll one channel to the left
func TestHorizontalKeys(t *testing.T) {
	m := utils.NewTestModel(120, 30)
	utils.LoadTestPattern(&m, 64)
	utils.SetChannels(&m, 32)

	view := &m.Tabs[tui.TabPattern].Pattern

	if view.MaxColOffset(m.Info.Channels) == 0 {
		t.Fatal("32 channels should overflow a 120 column body")
	}

	for _, tc := range []struct {
		keys []string
		want int
	}{
		{[]string{"l"}, 1},
		{[]string{"l", "l", "l"}, 4},
		{[]string{"h"}, 3},
		{[]string{"L", "shift+right"}, 3 + 2*tui.ChannelColumnWidth},
		{[]string{"H", "shift+left"}, 3},
		{[]string{"h", "h", "h", "h"}, 0}, // clamped at the start
	} {
		m = press(t, m, tc.keys...)

		if got := m.Tabs[tui.TabPattern].Pattern.ColOffset; got != tc.want {
			t.Errorf("%v: ColOffset = %d, want %d", tc.keys, got, tc.want)
		}
	}
}

// TestHorizontalKeysDoNotMoveTheSelection is the property that separates a scroll
// from a page: moving sideways changes the view, never the selection.
func TestHorizontalKeysDoNotMoveTheSelection(t *testing.T) {
	m := utils.NewTestModel(120, 30)
	utils.LoadTestPattern(&m, 64)
	utils.SetChannels(&m, 32)

	view := &m.Tabs[tui.TabPattern].Pattern
	view.Row = 20
	view.RowOffset = 10
	view.Clamp(m.Info.Channels)

	row, offset := view.Row, view.RowOffset

	m = press(t, m, "l", "l", "L", "h")

	if got := m.Tabs[tui.TabPattern].Pattern.Row; got != row {
		t.Errorf("scrolling moved the selection to row %d, want %d", got, row)
	}

	if got := m.Tabs[tui.TabPattern].Pattern.RowOffset; got != offset {
		t.Errorf("scrolling moved the row window to %d, want %d", got, offset)
	}
}

func TestTabKeyIsNotShadowedByTheLeader(t *testing.T) {
	m := utils.NewTestModel(100, 30)

	m = press(t, m, "tab")
	if m.ActiveTab != tui.TabInfo {
		t.Errorf("after tab the tab is %v, want info", m.ActiveTab)
	}

	m = press(t, m, "shift+tab")
	if m.ActiveTab != tui.TabPattern {
		t.Errorf("after shift+tab the tab is %v, want pattern", m.ActiveTab)
	}
}

func TestCommandLineTakesOverTheFooter(t *testing.T) {
	m := utils.NewTestModel(100, 30)

	m = press(t, m, ":")
	if !m.CommandActive {
		t.Fatal("the command line did not open")
	}

	// The prompt and the input share one row.
	if got := m.FooterLine(); got != m.CommandLine() {
		t.Error("the footer is not showing the command line while it is open")
	}

	// The input is reset once the command runs, so the effect has to be checked
	// on the model, not on what was typed.
	m = press(t, m, "t", "a", "b", " ", "i", "n", "f", "o", "enter")
	if m.CommandActive {
		t.Error("the command line did not close on enter")
	}

	if m.ActiveTab != tui.TabInfo {
		t.Errorf(":tab info left the tab at %v", m.ActiveTab)
	}
}

func TestEscapeClosesTheCommandLine(t *testing.T) {
	m := utils.NewTestModel(100, 30)

	m = press(t, m, ":", "q", "q", "esc")
	if m.CommandActive {
		t.Error("the command line did not close on escape")
	}

	if got := m.Command.Value(); got != "" {
		t.Errorf("command value = %q after escape, want it discarded", got)
	}
}

// keyPress builds a KeyPressMsg for a key name. The real handler matches on the
// string form, so the tests drive the same path the terminal does.
func keyPress(key string) (tea.KeyPressMsg, bool) {
	codes := map[string]tea.KeyPressMsg{
		"tab":         {Code: tea.KeyTab},
		"shift+tab":   {Code: tea.KeyTab, Mod: tea.ModShift},
		"shift+right": {Code: tea.KeyRight, Mod: tea.ModShift},
		"shift+left":  {Code: tea.KeyLeft, Mod: tea.ModShift},
		"esc":         {Code: tea.KeyEscape},
		"enter":       {Code: tea.KeyEnter},
		" ":           {Code: tea.KeySpace, Text: " "},
		tui.LeaderKey: {Code: '\\', Text: tui.LeaderKey},
	}

	if msg, ok := codes[key]; ok {
		return msg, true
	}

	// Anything else is a single printable rune.
	if len([]rune(key)) == 1 {
		return tea.KeyPressMsg{Code: rune(key[0]), Text: key}, true
	}

	return tea.KeyPressMsg{}, false
}
