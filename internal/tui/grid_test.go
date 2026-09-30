package tui_test

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/Lakelimbo/nezumi/internal/tui"
	"github.com/Lakelimbo/nezumi/internal/utils"
	"github.com/charmbracelet/x/ansi"
)

// TestRealModulesFillTheBody tests the shapes e2e
func TestRealModulesFillTheBody(t *testing.T) {
	for _, module := range utils.DemoModules(t) {
		t.Run(filepath.Base(module), func(t *testing.T) {
			m := utils.LoadModule(t, module)

			for _, size := range [][2]int{{80, 24}, {120, 30}, {200, 50}, {60, 20}, {20, 8}} {
				width, height := size[0], size[1]
				utils.Resize(&m, width, height)

				v := m.Tabs[tui.TabPattern].Pattern
				grid := m.PatternGrid()

				if got := lipgloss.Height(grid); got != v.ViewRows()+1 {
					t.Errorf("%dx%d: grid is %d rows, want %d",
						width, height, got, v.ViewRows()+1)
				}

				for i, line := range strings.Split(grid, "\n") {
					if got := lipgloss.Width(line); got != width {
						t.Errorf("%dx%d: line %d is %d columns, want %d",
							width, height, i, got, width)
					}
				}

				// the whole screen, not just the grid, is the contract
				if got := lipgloss.Height(m.View().Content); got != height {
					t.Errorf("%dx%d: view is %d rows, want %d",
						width, height, got, height)
				}
			}
		})
	}
}

// TestRealModuleScrollsAcrossEveryChannel walks the whole horizontal range of
// every demo module, so the scroll crosses several group separators
func TestRealModuleScrollsAcrossEveryChannel(t *testing.T) {
	for _, module := range utils.DemoModules(t) {
		t.Run(filepath.Base(module), func(t *testing.T) {
			m := utils.LoadModule(t, module)
			utils.Resize(&m, 120, 30)

			ch := m.Info.Channels
			view := &m.Tabs[tui.TabPattern].Pattern
			max := view.MaxColOffset(ch)

			for offset := 0; offset <= max; offset++ {
				view.ColOffset = offset
				view.Clamp(ch)

				// including the offsets that cut a channel in half
				if got := lipgloss.Width(m.PatternRow(view.Row, *view)); got != 120 {
					t.Fatalf("offset %d of %d: row is %d columns, want 120",
						offset, max, got)
				}
			}

			// the far end has to show the last channel
			view.ColOffset = max
			view.Clamp(ch)

			header, _ := utils.GridLines(m)
			if want := fmt.Sprintf("Ch %d", ch); !strings.Contains(ansi.Strip(header), want) {
				t.Errorf("at the far end the header does not name %q: %q",
					want, ansi.Strip(header))
			}
		})
	}
}

// TestRealModuleGroupSeparatorsHoldColumn is the property that made the old
// paged grid feel like it moved under you. The separators sit at fixed columns,
// so scrolling by one column shifts each visible separator by exactly one
func TestRealModuleGroupSeparatorsHoldColumn(t *testing.T) {
	for _, module := range utils.DemoModules(t) {
		if m := utils.LoadModule(t, module); m.Info.Channels <= tui.ChannelGroupSize*2 {
			continue
		}

		t.Run(filepath.Base(module), func(t *testing.T) {
			m := utils.LoadModule(t, module)
			utils.Resize(&m, 120, 30)

			view := &m.Tabs[tui.TabPattern].Pattern
			marker := utils.ForegroundSGR(m.Theme.Separator)

			// body column of each visible separator, to compare across offsets
			separatorColumns := func() []int {
				first, last := tui.VisibleChannels(
					view.ColOffset, view.Avail(), m.Info.Channels,
				)

				segs := m.ChannelSegments(
					m.Tabs[tui.TabPattern].Formatted[0][first:last+1], first, nil,
				)

				var cols []int

				col := first * tui.ChannelColumnWidth

				for _, seg := range segs {
					if strings.Contains(seg.Render(), marker) {
						cols = append(cols, col-view.ColOffset)
					}

					col += lipgloss.Width(seg.Text)
				}

				return cols
			}

			// every separator must land on a group boundary at every offset. Only
			// checking the un-scrolled window is not enough: there the first
			// visible channel is channel 1, so grouping relative to the window
			// and grouping relative to the pattern happen to agree.
			for offset := 0; offset <= view.MaxColOffset(m.Info.Channels); offset++ {
				view.ColOffset = offset
				view.Clamp(m.Info.Channels)

				for _, col := range separatorColumns() {
					abs := view.ColOffset + col

					if abs%tui.ChannelColumnWidth != tui.CellWidth {
						t.Errorf("offset %d: separator at column %d is not a tail column",
							offset, abs)
					}

					if ch := abs / tui.ChannelColumnWidth; ch%tui.ChannelGroupSize != 0 {
						t.Errorf(
							"offset %d: separator at column %d is in channel %d, want a multiple of %d",
							offset,
							abs,
							ch+1,
							tui.ChannelGroupSize,
						)
					}
				}
			}

			view.ColOffset = 0
			view.Clamp(m.Info.Channels)

			// scrolling shifts the whole set by exactly the offset
			before := separatorColumns()

			for offset := 1; offset <= view.MaxColOffset(m.Info.Channels); offset++ {
				view.ColOffset = offset
				view.Clamp(m.Info.Channels)

				after := separatorColumns()
				if len(after) != len(before) {
					continue // a separator entered or left the window
				}

				for i := range before {
					if want := before[i] - offset; after[i] != want {
						t.Fatalf("offset %d: separator %d is at column %d, want %d",
							offset, i, after[i], want)
					}
				}
			}
		})
	}
}

// TestRealModuleResizeSweepDoesNotPanic is the regression test for the original
// crash, driven through the real handler and real modules.
//
// The table indexed a row's cells and its columns in lockstep, so a resize that
// narrowed the columns while the old, wider rows were still installed indexed
// past the end of the column slice. Stepping a whole channel at a time hits
// every width where the number of fully visible channels changesd
func TestRealModuleResizeSweepDoesNotPanic(t *testing.T) {
	for _, module := range utils.DemoModules(t) {
		t.Run(filepath.Base(module), func(t *testing.T) {
			m := utils.LoadModule(t, module)
			utils.Resize(&m, 240, 60)

			view := &m.Tabs[tui.TabPattern].Pattern
			view.Row = min(20, view.Rows-1)
			view.ColOffset = view.MaxColOffset(m.Info.Channels) / 2
			view.Clamp(m.Info.Channels)

			step := tui.ChannelColumnWidth

			for width := 240; width >= 1; width -= step {
				utils.Resize(&m, width, 60)
				m.PatternGrid()
			}

			for width := 1; width <= 240; width += step {
				utils.Resize(&m, width, 60)
				m.PatternGrid()
			}

			// the narrowest bodies, where the row fill has to take over
			for _, width := range []int{
				tui.RowNumberWidth + tui.ChannelColumnWidth,
				tui.RowNumberWidth + 1,
				tui.RowNumberWidth,
				1,
			} {
				utils.Resize(&m, width, 60)
				m.PatternGrid()
			}

			// the row window is driven by the height, so it gets its own sweep
			for height := 1; height <= 60; height += 2 {
				utils.Resize(&m, 120, height)
				m.PatternGrid()
			}
		})
	}
}

// TestRealModuleBandIsStableAcrossResize: the highlighted row belongs to the
// pattern, so no resize may move it
func TestRealModuleBandIsStableAcrossResize(t *testing.T) {
	for _, module := range utils.DemoModules(t) {
		t.Run(filepath.Base(module), func(t *testing.T) {
			m := utils.LoadModule(t, module)
			utils.Resize(&m, 200, 40)

			view := &m.Tabs[tui.TabPattern].Pattern
			view.Row = min(20, view.Rows-1)
			view.Clamp(m.Info.Channels)

			row := view.Row
			marker := utils.BackgroundSGR(m.Theme.Highlight)

			for _, size := range [][2]int{{200, 40}, {80, 24}, {40, 12}, {12, 6}, {240, 60}} {
				utils.Resize(&m, size[0], size[1])
				view := m.Tabs[tui.TabPattern].Pattern

				if view.Row != row {
					t.Errorf("%dx%d: resize moved the band to row %d, want %d",
						size[0], size[1], view.Row, row)
				}

				grid := m.PatternGrid()
				if grid == "" {
					continue // too narrow to draw anything
				}

				// a body tall enough to show the band must show it
				if view.Rows > view.ViewRows() && !strings.Contains(grid, marker) {
					t.Errorf("%dx%d: the band is on screen but is not drawn", size[0], size[1])
				}
			}
		})
	}
}

// TestRealModuleFollowsPlayback drives the follow logic across a real pattern,
// including the jump from the last row back to the first
func TestRealModuleFollowsPlayback(t *testing.T) {
	for _, module := range utils.DemoModules(t) {
		t.Run(filepath.Base(module), func(t *testing.T) {
			m := utils.LoadModule(t, module)
			utils.Resize(&m, 120, 30)

			rows := m.Tabs[tui.TabPattern].Pattern.Rows
			if rows < 4 {
				t.Skip("pattern too short to follow")
			}

			for _, row := range []int{0, 1, rows / 2, rows - 1, 0, rows / 3} {
				m.Position.Row = row
				m.FollowPatternRow(row)

				view := m.Tabs[tui.TabPattern].Pattern

				if view.Row != min(row, rows-1) {
					t.Errorf("row %d: the band is on row %d", row, view.Row)
				}

				// the band has to be inside the window, or it is playing
				// somewhere the user cannot see
				if rows > view.ViewRows() &&
					(view.Row < view.RowOffset ||
						view.Row >= view.RowOffset+view.ViewRows()) {
					t.Errorf("row %d: the band at %d is outside the window %d..%d",
						row, view.Row, view.RowOffset, view.RowOffset+view.ViewRows())
				}
			}
		})
	}
}

// TestRealModuleHeaderIsAFaithfulCut checks the header scrolls with the grid
// rather than being re-pinned to whatever is visible.
func TestRealModuleHeaderIsAFaithfulCut(t *testing.T) {
	for _, module := range utils.DemoModules(t) {
		if m := utils.LoadModule(t, module); m.Info.Channels*tui.ChannelColumnWidth <= 115 {
			continue // fits in a 120 column body, nothing to scroll
		}

		t.Run(filepath.Base(module), func(t *testing.T) {
			m := utils.LoadModule(t, module)
			utils.Resize(&m, 120, 30)

			view := &m.Tabs[tui.TabPattern].Pattern
			ch := m.Info.Channels

			// the full label line, laid out once and cut at every offset
			var full strings.Builder
			fmt.Fprintf(&full, "%-*s", tui.RowNumberWidth, "#")

			for c := range ch {
				fmt.Fprintf(&full, "%-*s", tui.ChannelColumnWidth,
					" Ch "+strconv.Itoa(c+1))
			}

			plain := full.String()

			for offset := 0; offset <= view.MaxColOffset(ch); offset++ {
				view.ColOffset = offset
				view.Clamp(ch)

				header, _ := utils.GridLines(m)

				gutter := min(tui.RowNumberWidth, len(plain))
				want := tui.CutColumns(plain[gutter:], offset, offset+view.Avail())

				if got := ansi.Strip(header); !strings.HasPrefix(got[gutter:], want) {
					t.Fatalf("offset %d: header is %q, want it to start with %q",
						offset, got[gutter:], want)
				}
			}
		})
	}
}

// TestRealModuleRenderIsBounded is a guard on the per-frame work. The cost has
// to follow the size of the window rather than the size of the pattern
func TestRealModuleRenderIsBounded(t *testing.T) {
	for _, module := range utils.DemoModules(t) {
		t.Run(filepath.Base(module), func(t *testing.T) {
			m := utils.LoadModule(t, module)
			utils.Resize(&m, 200, 50)

			// one warm-up frame, so first-touch page faults are not measured
			m.PatternGrid()

			const frames = 50
			allocs := testing.AllocsPerRun(frames, func() { m.PatternGrid() })

			// a frame lays out one segment per visible field, so a few thousand
			// allocations is the expected order
			const limit = 40_000.0

			if allocs > limit {
				t.Errorf("a frame allocates %.0f times, want under %.0f -- "+
					"the render is no longer bounded by the window", allocs, limit)
			}

			// the count must not depend on the channel count
			narrow := m
			utils.Resize(&narrow, 40, 50)
			narrow.PatternGrid()

			before := testing.AllocsPerRun(frames, func() { m.PatternGrid() })
			after := testing.AllocsPerRun(frames, func() { narrow.PatternGrid() })

			// a 40 column body shows a fifth of the channels
			if after > before {
				t.Errorf("a 40 column body allocates %.0f times, more than the "+
					"200 column body's %.0f", after, before)
			}
		})
	}
}

// TestChannelReadoutNamesTheVisibleRange checks the footer says which channels
// are on screen. Without it a user would have to count group separators
func TestChannelReadoutNamesTheVisibleRange(t *testing.T) {
	var m tui.Model

	for _, module := range utils.DemoModules(t) {
		if candidate := utils.LoadModule(t, module); candidate.Info.Channels > 8 {
			m = candidate
			break
		}
	}

	if m.Info.Channels == 0 {
		t.Skip("no module with more than 8 channels in demo/")
	}

	utils.Resize(&m, 120, 30)

	view := &m.Tabs[tui.TabPattern].Pattern
	total := m.Info.Channels

	if view.MaxColOffset(total) == 0 {
		t.Fatalf("%d channels fit in a 120 column body; the test needs one that does not",
			total)
	}

	// a body wide enough for every channel has nothing to report
	m.Width = total*tui.ChannelColumnWidth + tui.RowNumberWidth
	view.Width = m.Width
	view.Clamp(total)

	if got := m.ChannelReadout(); got != "" {
		t.Errorf("readout = %q with the whole grid on screen, want empty", got)
	}

	view.Width = 120
	view.Clamp(total)

	for offset := 0; offset <= view.MaxColOffset(total); offset++ {
		view.ColOffset = offset
		view.Clamp(total)

		first, last := tui.VisibleChannels(offset, view.Avail(), total)
		want := fmt.Sprintf("ch %d-%d/%d", first+1, min(last, total-1)+1, total)

		if got := m.ChannelReadout(); got != want {
			t.Errorf("offset %d: readout = %q, want %q", offset, got, want)
		}
	}
}

// TestRealModuleRowsAreLabelled checks the gutter is the row number
func TestRealModuleRowsAreLabelled(t *testing.T) {
	for _, module := range utils.DemoModules(t) {
		t.Run(filepath.Base(module), func(t *testing.T) {
			m := utils.LoadModule(t, module)
			utils.Resize(&m, 120, 30)

			view := &m.Tabs[tui.TabPattern].Pattern
			view.Row = min(20, view.Rows-1)
			view.Clamp(m.Info.Channels)

			_, rows := utils.GridLines(m)

			for i, row := range rows {
				want := fmt.Sprintf("%-*d", tui.RowLabelNumberWidth, view.RowOffset+i)
				if !strings.Contains(ansi.Strip(row), want) {
					t.Errorf("row line %d does not carry the label %q", i, want)
				}
			}
		})
	}
}

// TestChannelCountsAreCovered records the shapes the renderer is expected to
// handle, so a new format with a count outside this set gets looked at
func TestChannelCountsAreCovered(t *testing.T) {
	for _, module := range utils.DemoModules(t) {
		m := utils.LoadModule(t, module)

		switch ch := m.Info.Channels; {
		case ch < 1:
			t.Errorf("%s reports %d channels", filepath.Base(module), ch)
		case ch > 64:
			t.Errorf("%s reports %d channels, past the widest layout that is tested",
				filepath.Base(module), ch)
		}

		grid := m.Tabs[tui.TabPattern].Formatted
		if len(grid) == 0 {
			continue
		}

		// the decoded grid has to agree with the reported count, or the scroll
		// arithmetic is working off the wrong number
		if got := len(grid[0]); got != m.Info.Channels {
			t.Errorf("%s: the first row has %d cells but the module reports %d channels",
				filepath.Base(module), got, m.Info.Channels)
		}
	}
}

// TestModuleListIsStable keeps the demo set from silently shrinking, which would
// reduce the coverage of every test above
func TestModuleListIsStable(t *testing.T) {
	modules := utils.DemoModules(t)

	for _, want := range []string{
		"4mat-world_of_dentro.mod",
		"dubmood-3d_galax.xm",
		"necros-realization-ii.s3m",
		"nightbeat-dreamstone.it",
		"siren-carpe_diem.it",
	} {
		if !slices.ContainsFunc(modules, func(p string) bool {
			return filepath.Base(p) == want
		}) {
			t.Errorf("demo/%s is missing", want)
		}
	}
}
