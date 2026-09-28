package utils

import (
	"os"
	"path/filepath"
	"testing"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/Lakelimbo/nezumi/internal/tui"
	"github.com/Lakelimbo/nezumi/libopenmpt"
)

// NewTestModel builds a Model with no module and no audio device. It goes
// through NewTabs like New does, so the component setup cannot drift; the tests
// supply only the metadata and the pattern to draw
func NewTestModel(width, height int) tui.Model {
	command := textinput.New()
	command.Placeholder = "command"

	m := tui.Model{
		ActiveTab:     tui.TabPattern,
		LoadedPattern: -1,
		Playing:       true,
		Command:       command,
		Tabs:          tui.NewTabs(),
		Theme:         tui.DefaultTheme(),
		Info: tui.ModuleInfo{
			Title:    "test module",
			Artist:   "nobody",
			Tracker:  "FastTracker II",
			TypeLong: "Extended Module",
			Duration: 64 * 1e9,
			Channels: 4,
			Patterns: 8,
		},
	}

	Resize(&m, width, height)

	return m
}

// Resize drives a window-size change through the real handler, which is the path
// the terminal drives and the one the crash came through
func Resize(m *tui.Model, width, height int) {
	updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})

	if next, ok := updated.(tui.Model); ok {
		*m = next
	}
}

// LoadTestPattern gives m a pattern with data on the first channel of every
// fourth row, which is the shape most modules have and leaves the rest empty
// for the renderer to mute
func LoadTestPattern(m *tui.Model, rows int) {
	pattern := libopenmpt.PatternData{Index: 0, Rows: rows}
	pattern.Cells = make([][]libopenmpt.Cell, rows)

	for r := range pattern.Cells {
		pattern.Cells[r] = make([]libopenmpt.Cell, m.Info.Channels)

		if r%4 == 0 {
			pattern.Cells[r][0] = libopenmpt.Cell{
				Note:       uint8(49 + r%12),
				Instrument: 1,
				Volume:     40,
				Effect:     1,
				Parameter:  6,
			}
		}
	}

	m.LoadedPattern = 0
	m.LoadPattern(pattern)

	// the view now knows how many rows there are, so the window has to be
	// refitted against them
	Resize(m, m.Width, m.Height)
}

// SetChannels gives m a different channel count, as a wider module would. The
// fixture is rebuilt, so the decoded grid always has as many cells per row as
// the channel header claims
func SetChannels(m *tui.Model, channels int) {
	rows := m.Pattern.Rows
	m.Info.Channels = channels

	LoadTestPattern(m, rows)
}

// DemoModules lists the modules in demo/
func DemoModules(t *testing.T) []string {
	t.Helper()

	root := filepath.Join("..", "..", "demo")

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Skipf("no demo directory: %v", err)
	}

	var modules []string

	for _, entry := range entries {
		if !entry.IsDir() {
			modules = append(modules, filepath.Join(root, entry.Name()))
		}
	}

	return modules
}

// LoadModule opens a demo module and decodes its first pattern into the grid
// the renderer draws from. Tests that skip rather than fail on an unopenable
// module, since the demo set is data, not part of the build
func LoadModule(t *testing.T, path string) tui.Model {
	t.Helper()

	mod, err := libopenmpt.Open(path)
	if err != nil {
		t.Skipf("cannot open %s: %v", filepath.Base(path), err)
	}

	t.Cleanup(func() { mod.Close() })

	m := NewTestModel(0, 0)
	m.Module = mod
	m.Info = tui.ModuleInfo{
		Title:    mod.Title(),
		TypeLong: mod.TypeLong(),
		Channels: mod.NumChannels(),
		Patterns: mod.NumPatterns(),
	}

	data, err := mod.ReadPattern(0)
	if err != nil {
		t.Skipf("cannot read pattern 0 of %s: %v", filepath.Base(path), err)
	}

	m.LoadPattern(data)
	Resize(&m, m.Width, m.Height)

	return m
}
