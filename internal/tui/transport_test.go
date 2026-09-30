package tui_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Lakelimbo/nezumi/internal/tui"
	"github.com/Lakelimbo/nezumi/internal/utils"
	"github.com/Lakelimbo/nezumi/libopenmpt"
)

// TestFollowOffTabState tests whether the tab is the one where
// the user left if follow is off
func TestFollowOffTabState(t *testing.T) {
	m, _ := newTransportModel(t)
	loadOrder(t, &m, 0, 12)

	if m.LoadedOrder != 0 {
		t.Errorf("LoadedOrder is %d with follow off, want 0", m.LoadedOrder)
	}

	if got := m.Tabs[tui.TabPattern].Pattern.Row; got != 12 {
		t.Errorf("cursor is at row %d with follow off, want the 12 it was left on", got)
	}
}

func TestFollowOnAdoptsBandOrder(t *testing.T) {
	m, mod := newTransportModel(t)
	loadOrder(t, &m, 0, 12)

	m.Tabs[tui.TabPattern].Follow = true

	cmd := m.FollowPosition()

	if m.LoadedOrder != 14 {
		t.Errorf("LoadedOrder is %d with follow on, want 14", m.LoadedOrder)
	}

	if got := m.Tabs[tui.TabPattern].Pattern.Row; got != 0 {
		t.Errorf("cursor is at row %d with follow on, want the band's 0", got)
	}

	if cmd == nil {
		t.Error("following into a new order returned no command, so the pattern never loads")
	}

	// order 14 holds pattern 12, so a shim returning the order would ask for the
	// wrong grid
	if got := mod.OrderPattern(14); got != 12 {
		t.Fatalf("fixture assumption broken: order 14 holds pattern %d, want 12", got)
	}
}

func TestFollowOnReusesProperLoadedOrder(t *testing.T) {
	m, _ := newTransportModel(t)
	loadOrder(t, &m, 0, 4)

	m.Tabs[tui.TabPattern].Follow = true
	m.FollowPosition()

	// still order 0, so following only has to move the cursor
	if m.LoadedOrder != 0 {
		t.Errorf("LoadedOrder is %d, want 0", m.LoadedOrder)
	}
}

func TestFollowOffLoadPatternCursorKept(t *testing.T) {
	m, _ := newTransportModel(t)
	loadOrder(t, &m, 0, 12)
	m.Tabs[tui.TabPattern].Follow = false

	index := m.Module.OrderPattern(0)
	m = update(t, m, tui.PatternMsg{
		Index: index,
		Data:  readPattern(t, m.Module, index),
	})

	if got := m.Tabs[tui.TabPattern].Pattern.Row; got != 12 {
		t.Errorf("cursor is at row %d after reloading pattern, want 12", got)
	}
}

func TestLoadPatternRecenter(t *testing.T) {
	m, _ := newTransportModel(t)
	loadOrder(t, &m, 0, 12)

	m.Tabs[tui.TabPattern].Follow = true
	m.Position.Row = 8

	index := m.Module.OrderPattern(0)
	m = update(t, m, tui.PatternMsg{
		Index: index,
		Data:  readPattern(t, m.Module, index),
	})

	if got := m.Tabs[tui.TabPattern].Pattern.Row; got != 8 {
		t.Errorf("cursor is at %d after reloading, want the band's 8", got)
	}
}

func TestSeekRejection(t *testing.T) {
	m, mod := newTransportModel(t)
	loadOrder(t, &m, 0, 12)

	for _, order := range []int{-1, mod.NumOrders()} {
		m.Status = ""

		if cmd := m.SeekTo(order, 0); cmd != nil {
			t.Errorf("SeekTo(%d) returned a command, but the order should've been refused", order)
		}

		if m.Status == "" {
			t.Errorf("SeekTo(%d) refused the order without outputting a status", order)
		}

		if got := mod.CurrentOrder(); got != 0 {
			t.Errorf("the module moved to order %d despite both seeks being rejected", got)
		}

		if got := m.Tabs[tui.TabPattern].Pattern.Row; got != 12 {
			t.Errorf("the cursor moved tor ow %d despite both seeks being rejected", got)
		}
	}
}

func TestAdjacentOrdersLanding(t *testing.T) {
	m, mod := newTransportModel(t)

	for order := range mod.NumOrders() {
		for _, step := range []int{1, -1} {
			if next := m.AdjacentOrder(order, step); !mod.IsOrderPlayable(next) {
				t.Errorf("AdjacentOrder(%d, %d) is %d, which is not a playablwe order",
					order, step, next)
			}
		}
	}
}

func TestAdjacentOrderWrap(t *testing.T) {
	m, mod := newTransportModel(t)
	total := mod.NumOrders()

	last := -1
	for order := range total {
		if mod.IsOrderPlayable(order) {
			last = order
		}
	}
	if last < 0 {
		t.Skip("module has no playable order")
	}

	if next := m.AdjacentOrder(last, 1); next > last {
		t.Errorf("advancing past the last playable order (%d) gave %d, want a wrap",
			last, next)
	}

	first := -1
	for order := range total {
		if mod.IsOrderPlayable(order) {
			first = order
			break
		}
	}

	if next := m.AdjacentOrder(first, -1); next >= 0 && next < first {
		t.Errorf("backing up before the first playable order (%d) gave %d, want a wrap",
			first, next)
	}
}

func TestCursorTransportPlaybackOff(t *testing.T) {
	m := tui.Model{}

	if !m.CursorIsTransport() {
		t.Error("cursor is not the transport while stopped")
	}

	m.Playing = true
	if m.CursorIsTransport() {
		t.Error("cursor is the transport while the band is running")
	}
}

func TestParseSeek(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		order int
		row   int
		ok    bool
	}{
		{[]string{"4"}, 4, 0, true},
		{[]string{"4", "16"}, 4, 16, true},
		{[]string{"0"}, 0, 0, true},
		{[]string{"-1"}, -1, 0, true}, //parses, then SeekTo rejects
		{nil, 4, 0, false},
		{[]string{}, 0, 0, false},
		{[]string{"4", "16", "2"}, 0, 0, false},
		{[]string{"x"}, 0, 0, false},
		{[]string{"4", "x"}, 0, 0, false},
	} {
		order, row, ok := tui.ParseSeek(tc.args)

		if ok != tc.ok {
			t.Errorf("ParseSeek(%q) ok is %t, want %t", tc.args, ok, tc.ok)
		}

		if ok && (order != tc.order || row != tc.row) {
			t.Errorf("ParseSeek(%q) is %d:%d, want %d:%d",
				tc.args, order, row, tc.order, tc.row)
		}
	}
}

// -- helpers --

func newTransportModel(t *testing.T) (tui.Model, *libopenmpt.Module) {
	t.Helper()

	mod, err := libopenmpt.Open(string(utils.DemoRealization))
	if err != nil {
		t.Skipf("cannot open the module: %v", err)
	}

	t.Cleanup(mod.Close)

	m := tui.Model{
		Module:      mod,
		ActiveTab:   tui.TabPattern,
		LoadedOrder: -1,
		Tabs:        tui.NewTabs(),
		Theme:       tui.DefaultTheme(),
		Info: tui.ModuleInfo{
			Channels: mod.NumChannels(),
			Patterns: mod.NumPatterns(),
		},
	}

	return m, mod
}

func loadOrder(t *testing.T, m *tui.Model, order, row int) {
	t.Helper()

	m.LoadedOrder = order
	m.LoadPattern(readPattern(t, m.Module, m.Module.OrderPattern(order)))
	m.Tabs[tui.TabPattern].Pattern.CentreOn(row, m.Info.Channels)
}

func readPattern(t *testing.T, mod *libopenmpt.Module, index int) libopenmpt.PatternData {
	t.Helper()

	data, err := mod.ReadPattern(index)
	if err != nil {
		t.Fatalf("ReadPattern(%d): %v", data, err)
	}

	return data
}

func update(t *testing.T, m tui.Model, msg tea.Msg) tui.Model {
	t.Helper()

	updated, _ := m.Update(msg)

	next, ok := updated.(tui.Model)
	if !ok {
		t.Fatalf("update returned %T, want tui.Model", updated)
	}

	return next
}
