package tui_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/Lakelimbo/nezumi/internal/tui"
	"github.com/Lakelimbo/nezumi/internal/utils"
	"github.com/Lakelimbo/nezumi/libopenmpt"
)

// TestPatternGridWholeBody tests whether the pattern properly fills the body
func TestPatternGridWholeBody(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {120, 30}, {200, 50}, {40, 10}} {
		width, height := size[0], size[1]

		m := utils.NewTestModel(width, height)
		utils.LoadTestPattern(&m, 64)

		v := m.Tabs[tui.TabPattern].Pattern
		_, rows := utils.GridLines(m)

		if want := v.ViewRows(); len(rows) != want {
			t.Errorf("%dx%d: %d row lines, want %d", width, height, len(rows), want)
		}

		for i, row := range rows {
			if got := lipgloss.Width(row); got != width {
				t.Errorf("%dx%d: row %d is %d columns, want %d",
					width, height, i, got, width)
			}
		}

		if got := lipgloss.Width(m.PatternGrid()); got != width {
			t.Errorf("%dx%d: grid is %d columns, want %d", width, height, got, width)
		}
	}
}

// TestCellSegmentsFixedWidth tests whether each cell is exactly as wide as the
// rendered field, so a cell that overflows cannot desync the scroll
func TestCellSegmentsFixedWidth(t *testing.T) {
	m := utils.NewTestModel(120, 30)
	utils.LoadTestPattern(&m, 64)

	want := map[string]int{
		"note":       tui.NoteFieldWidth,
		"instrument": tui.ValueFieldWidth,
		"volume":     tui.ValueFieldWidth,
		"effect":     tui.EffectFieldWidth,
	}

	// boundary cells carry the separator split across the two tail columns, so
	// they are checked as a total rather than field by field
	for _, boundary := range []bool{false, true} {
		segs := m.CellSegments(tui.CellFields{
			Note:       "C-4",
			Instrument: "01",
			Volume:     "40",
			Effect:     "106",
		}, nil, boundary)

		if got := utils.SegmentWidth(segs); got != tui.ChannelColumnWidth {
			t.Errorf("boundary=%v: cell is %d columns, want %d",
				boundary, got, tui.ChannelColumnWidth)
		}

		for i, name := range []string{"note", "instrument", "volume", "effect"} {
			if got := lipgloss.Width(segs[i].Text); got != want[name] {
				t.Errorf("boundary=%v: %s field is %d columns, want %d",
					boundary, name, got, want[name])
			}
		}
	}
}

// TestRowNumberGutterPinned tests behavior of the gutter, which is not part of
// horizontal scrolling
func TestRowNumberGutterPinned(t *testing.T) {
	m := utils.NewTestModel(120, 30)
	utils.LoadTestPattern(&m, 64)

	want := m.PatternRow(0, m.Tabs[tui.TabPattern].Pattern)

	for offset := range 8 {
		view := &m.Tabs[tui.TabPattern].Pattern
		view.ColOffset = offset
		view.Clamp(m.Info.Channels)

		got := m.PatternRow(0, *view)
		if lipgloss.Width(got) != lipgloss.Width(want) {
			t.Errorf("offset %d: row is %d columns, want %d",
				offset, lipgloss.Width(got), lipgloss.Width(want))
		}
	}
}

// TestRowBandFullWidth tests whether the band has to be carried by each segment,
// since styled strings get resetted and the original style lost
func TestRowBandFullWidth(t *testing.T) {
	m := utils.NewTestModel(120, 30)
	utils.LoadTestPattern(&m, 64)

	// PatternGrid reads the row back off the tab state, so set it there
	m.Tabs[tui.TabPattern].Pattern.Row = 4
	v := m.Tabs[tui.TabPattern].Pattern

	marker := utils.BackgroundSGR(m.Theme.Highlight)
	_, rows := utils.GridLines(m)

	banded := v.Row - v.RowOffset
	if banded < 0 || banded >= len(rows) {
		t.Fatalf("selected row %d is outside the window %d..%d",
			v.Row, v.RowOffset, v.RowOffset+len(rows))
	}

	if !strings.Contains(rows[banded], marker) {
		t.Errorf("the selected row carries no band: %q", rows[banded])
	}

	for i, row := range rows {
		if i == banded {
			continue
		}

		if strings.Contains(row, marker) {
			t.Errorf("row %d is banded but is not the selection: %q", i, row)
		}
	}
}

// TestBandNarrowBodies tests the band at certain widths where the channel
// window no longer divides the body
func TestBandNarrowBodies(t *testing.T) {
	for _, width := range []int{20, 37, 64, 120} {
		m := utils.NewTestModel(width, 30)
		utils.LoadTestPattern(&m, 64)

		v := m.Tabs[tui.TabPattern].Pattern
		v.Row = v.RowOffset

		row := m.PatternRow(v.Row, v)

		if got := lipgloss.Width(row); got != width {
			t.Errorf("width %d: banded row is %d columns, want %d", width, got, width)
		}

		if !strings.Contains(row, utils.BackgroundSGR(m.Theme.Highlight)) {
			t.Errorf("width %d: no band on the selected row", width)
		}
	}
}

// TestHeaderNamesChannelsInPlace tests whether the channel names are aligned
// over the grid and also cut to the scroll window
func TestHeaderNamesChannelsInPlace(t *testing.T) {
	m := utils.NewTestModel(200, 30)
	utils.LoadTestPattern(&m, 64)
	utils.SetChannels(&m, 32)

	view := &m.Tabs[tui.TabPattern].Pattern
	header, _ := utils.GridLines(m)

	if !strings.Contains(header, "Ch 1") {
		t.Fatalf("header does not name the first channel: %q", header)
	}

	// bigger modules (wider pattern) have to be scrollable
	if view.MaxColOffset(m.Info.Channels) == 0 {
		t.Fatal("a 32 channel module should overflow a 200 column body")
	}

	view.ColOffset = view.MaxColOffset(m.Info.Channels)
	view.Clamp(m.Info.Channels)

	header, _ = utils.GridLines(m)

	if strings.Contains(header, "Ch 1 ") {
		t.Errorf("scrolled header still claims to start at channel 1: %q", header)
	}

	if !strings.Contains(header, "Ch 32") {
		t.Errorf("scrolled header does not reach the last channel: %q", header)
	}
}

// TestScrollColsMovesOneCell tests horizontal scrolling is by the column, so a
// channel can be cut in half
func TestScrollColsMovesOneCell(t *testing.T) {
	m := utils.NewTestModel(120, 30)
	utils.LoadTestPattern(&m, 64)
	utils.SetChannels(&m, 32)

	view := &m.Tabs[tui.TabPattern].Pattern
	view.ColOffset = 0

	for range 3 {
		view.ScrollCols(1, m.Info.Channels)
	}

	if view.ColOffset != 3 {
		t.Errorf("after three right presses ColOffset = %d, want 3", view.ColOffset)
	}

	view.ScrollCols(-1, m.Info.Channels)

	if view.ColOffset != 2 {
		t.Errorf("after one left press ColOffset = %d, want 2", view.ColOffset)
	}
}

// TestScrollColsStaysInBounds check whether the left and right ends have to be
// pulled so the last channel's edge lines up with the body's edge
func TestScrollColsStaysInBounds(t *testing.T) {
	m := utils.NewTestModel(120, 30)
	utils.LoadTestPattern(&m, 64)
	utils.SetChannels(&m, 32)

	view := &m.Tabs[tui.TabPattern].Pattern
	max := view.MaxColOffset(m.Info.Channels)

	if max == 0 {
		t.Fatal("a 32 channel module should overflow a 120 column body")
	}

	view.ScrollCols(-5, m.Info.Channels)

	if view.ColOffset != 0 {
		t.Errorf("scrolled left past the start: ColOffset = %d", view.ColOffset)
	}

	for range max + 50 {
		view.ScrollCols(1, m.Info.Channels)
	}

	if view.ColOffset != max {
		t.Errorf("ColOffset = %d, want the far end %d", view.ColOffset, max)
	}
}

// TestScrollDoesNotMoveSelection tests whether scrolling horizontally does not
// change the selection
func TestScrollDoesNotMoveSelection(t *testing.T) {
	m := utils.NewTestModel(120, 30)
	utils.LoadTestPattern(&m, 64)
	utils.SetChannels(&m, 32)

	view := &m.Tabs[tui.TabPattern].Pattern
	view.Row = 20
	view.RowOffset = 10

	row, offset := view.Row, view.RowOffset

	for range 40 {
		view.ScrollCols(1, m.Info.Channels)
	}

	if view.Row != row {
		t.Errorf("scrolling moved the selection to row %d, want %d", view.Row, row)
	}

	if view.RowOffset != offset {
		t.Errorf("scrolling moved the row window to %d, want %d", view.RowOffset, offset)
	}
}

// TestResizeDownKeepsSelection tests whether resizing keeps the selection
// without panicking
func TestResizeDownKeepsSelection(t *testing.T) {
	for _, width := range []int{200, 120, 60, 30, 12, 6, 200} {
		m := utils.NewTestModel(width, 30)
		utils.LoadTestPattern(&m, 64)

		view := &m.Tabs[tui.TabPattern].Pattern
		view.Row = 40
		view.Clamp(m.Info.Channels)

		row := view.Row

		// render at every width on the way down, not just at the end
		m.PatternGrid()

		if view.Row != row {
			t.Errorf("width %d: resize moved the selection to row %d, want %d",
				width, view.Row, row)
		}
	}
}

// TestResizeOnScreenScroll tests the direction the pattern has to be clamped if
// the window is narrower than the body
func TestResizeOnScreenScroll(t *testing.T) {
	m := utils.NewTestModel(200, 30)
	utils.LoadTestPattern(&m, 64)
	utils.SetChannels(&m, 32)

	view := &m.Tabs[tui.TabPattern].Pattern
	view.ColOffset = view.MaxColOffset(m.Info.Channels)
	utils.Resize(&m, 200, 30)

	if view.ColOffset != view.MaxColOffset(m.Info.Channels) {
		t.Errorf("ColOffset = %d, want it clamped to %d",
			view.ColOffset, view.MaxColOffset(m.Info.Channels))
	}

	// even though nobody may use such a small window, we should not crash
	utils.Resize(&m, 1, 30)

	if view.ColOffset < 0 {
		t.Errorf("ColOffset = %d at a 1 column body, want at least 0", view.ColOffset)
	}

	if m.PatternGrid() != "" {
		t.Errorf("a 1 column body should draw nothing, got %q", m.PatternGrid())
	}
}

// TestGroupSeparatorsAnchored tests whether the separators are placed on the
// absolute channel boundaries
func TestGroupSeparatorsAnchored(t *testing.T) {
	m := utils.NewTestModel(120, 30)
	utils.LoadTestPattern(&m, 64)
	utils.SetChannels(&m, 32)

	marker := utils.ForegroundSGR(m.Theme.Separator)
	segs := m.ChannelSegments(m.Tabs[tui.TabPattern].Formatted[0], 0, nil)

	col, found := 0, 0

	for _, seg := range segs {
		if strings.Contains(seg.Render(), marker) {
			found++

			// the bar is the first column of a channel's two column tail, so it
			// has to land on a fixed column of the grid
			if want := tui.CellWidth; col%tui.ChannelColumnWidth != want {
				t.Errorf("separator at column %d, want the first tail column %d",
					col, want)
			}

			if ch := col / tui.ChannelColumnWidth; ch%tui.ChannelGroupSize != 0 {
				t.Errorf("separator in channel %d, want one every %d channels",
					ch+1, tui.ChannelGroupSize)
			}
		}

		col += lipgloss.Width(seg.Text)
	}

	if found == 0 {
		t.Fatal("no group separators were drawn")
	}
}

// TestVisibleChannelsCover tests whether the range of the rendered pattern is
// boundedd by the channels the window touches
func TestVisibleChannelsCover(t *testing.T) {
	const channels = 32

	// widths expressed in no. of channels
	for _, whole := range []int{0, 1, 6, 13, channels, channels + 5} {
		avail := whole*tui.ChannelColumnWidth + 3 // always a partial channel over

		for offset := range 4 * tui.ChannelColumnWidth {
			first, last := tui.VisibleChannels(offset, avail, channels)

			if first > last || first < 0 || last >= channels {
				t.Fatalf("avail %d offset %d: range %d..%d is not usable",
					avail, offset, first, last)
			}

			// the first channel has to start at or before the window
			if first*tui.ChannelColumnWidth > offset {
				t.Errorf("avail %d offset %d: range starts at column %d, past the window",
					avail, offset, first*tui.ChannelColumnWidth)
			}

			// the last one has to reach the end of the window, unless the
			// window is already wider than the pattern
			if end := (last + 1) * tui.ChannelColumnWidth; end < offset+avail &&
				avail < channels*tui.ChannelColumnWidth {
				t.Errorf("avail %d offset %d: range ends at column %d, short of the window",
					avail, offset, end)
			}
		}
	}
}

// TestFollowPatternRowCentresThePlayingRow checks that following playback moves
// the band, not just the window
func TestFollowPatternRowCentresThePlayingRow(t *testing.T) {
	m := utils.NewTestModel(120, 30)
	utils.LoadTestPattern(&m, 64)

	for _, playing := range []int{0, 1, 20, 40, 63} {
		m.FollowPatternRow(playing)

		view := m.Tabs[tui.TabPattern].Pattern

		if view.Row != min(max(playing, 0), 63) {
			t.Errorf("FollowPatternRow(%d) selected row %d", playing, view.Row)
		}

		if view.Row < view.RowOffset || view.Row >= view.RowOffset+view.ViewRows() {
			t.Errorf("FollowPatternRow(%d): row %d outside window %d..%d",
				playing, view.Row, view.RowOffset, view.RowOffset+view.ViewRows())
		}
	}
}

// TestFollowPatternRowClampsToPattern tests if the selected row properly clamps
// to the pattern grid
func TestFollowPatternRowClampsToPattern(t *testing.T) {
	m := utils.NewTestModel(120, 30)
	utils.LoadTestPattern(&m, 64)

	for _, row := range []int{-10, 0, 63, 5000} {
		m.FollowPatternRow(row)

		view := m.Tabs[tui.TabPattern].Pattern

		if want := min(max(row, 0), 63); view.Row != want {
			t.Errorf("FollowPatternRow(%d) selected row %d, want %d", row, view.Row, want)
		}
	}
}

// TestFollowKeepsTheHorizontalScroll tests whether the playback does not reset
// ("jump") the horizontal scroll offset
func TestFollowKeepsTheHorizontalScroll(t *testing.T) {
	m := utils.NewTestModel(120, 30)
	utils.LoadTestPattern(&m, 64)
	utils.SetChannels(&m, 32)

	view := &m.Tabs[tui.TabPattern].Pattern
	view.ColOffset = 120
	m.FollowPatternRow(30)

	if view.ColOffset != 120 {
		t.Errorf("follow moved the column scroll to %d, want it left at 120", view.ColOffset)
	}
}

// TestFollowIgnoresEmptyPattern tests if the view does not break before any
// pattern has been loaded and rendered
func TestFollowIgnoresEmptyPattern(t *testing.T) {
	m := utils.NewTestModel(120, 30)

	m.FollowPatternRow(12)

	if got := m.Tabs[tui.TabPattern].Pattern.Row; got != 0 {
		t.Errorf("Row = %d on an empty pattern, want 0", got)
	}

	if m.PatternGrid() == "" {
		t.Error("the channel header should still draw before a pattern loads")
	}
}

// TestRowNavigationClamps checks for vertical bounds and bodies larger than the
// pattern
func TestRowNavigationClamps(t *testing.T) {
	m := utils.NewTestModel(120, 100)
	utils.LoadTestPattern(&m, 8)

	view := &m.Tabs[tui.TabPattern].Pattern

	if view.ViewRows() <= 8 {
		t.Fatalf("body fits %d rows, want more than the pattern's 8", view.ViewRows())
	}

	view.GotoBottom(m.Info.Channels)

	if view.Row != 7 {
		t.Errorf("GotoBottom selected row %d, want 7", view.Row)
	}

	if view.RowOffset != 0 {
		t.Errorf("RowOffset = %d for a pattern shorter than the body, want 0",
			view.RowOffset)
	}

	view.GotoTop(m.Info.Channels)

	if view.Row != 0 || view.RowOffset != 0 {
		t.Errorf("GotoTop left row %d offset %d, want 0 and 0", view.Row, view.RowOffset)
	}
}

// TestVerticalScrollVisibleSelection: the selection moves with the window.
func TestVerticalScrollVisibleSelection(t *testing.T) {
	m := utils.NewTestModel(120, 30)
	utils.LoadTestPattern(&m, 64)

	view := &m.Tabs[tui.TabPattern].Pattern
	page := view.ViewRows()

	for range 4 {
		view.MoveRow(page, m.Info.Channels)

		if view.Row < view.RowOffset || view.Row >= view.RowOffset+page {
			t.Fatalf("row %d outside window %d..%d after paging",
				view.Row, view.RowOffset, view.RowOffset+page)
		}
	}

	for range 4 {
		view.MoveRow(-page, m.Info.Channels)

		if view.Row < view.RowOffset || view.Row >= view.RowOffset+page {
			t.Fatalf("row %d outside window %d..%d after paging back",
				view.Row, view.RowOffset, view.RowOffset+page)
		}
	}
}

// TestMutedEmptyFields checks the color rendering for different data in the
// pattern
func TestMutedEmptyFields(t *testing.T) {
	m := utils.NewTestModel(120, 30)
	utils.LoadTestPattern(&m, 64)

	// row 0 of the fixture has a full cell on channel 1; row 1 is empty
	grid := m.Tabs[tui.TabPattern].Formatted

	filled := m.CellSegments(grid[0][0], nil, false)
	empty := m.CellSegments(grid[1][0], nil, false)

	if !strings.Contains(filled[0].Render(), utils.ForegroundSGR(m.Theme.Note)) {
		t.Errorf("a cell with a note is not drawn in the note color: %q", filled[0].Text)
	}

	if strings.Contains(empty[0].Render(), utils.ForegroundSGR(m.Theme.Note)) {
		t.Errorf("an empty cell is drawn in the note color: %q", empty[0].Text)
	}

	if !strings.Contains(empty[0].Render(), utils.ForegroundSGR(m.Theme.Empty)) {
		t.Errorf("an empty cell is not drawn in the empty color: %q", empty[0].Text)
	}
}

// TestFourColumnEffectsDoNotShiftTheRow: commands are usually one hex digit and
// two above 0x0F, so "%X%02X" is either 3 or 4 columns, and sizing the field for
// the common case made the whole row shift sideways.
func TestFourColumnEffectsDoNotShiftTheRow(t *testing.T) {
	m := utils.NewTestModel(120, 30)
	utils.LoadTestPattern(&m, 64)

	// one-digit and two-digit commands, so both widths sit in the same row
	oneDigit := tui.ParseCell(libopenmpt.Cell{Note: 49, Effect: 4, Parameter: 0x0A})
	twoDigit := tui.ParseCell(libopenmpt.Cell{Note: 49, Effect: 0x12, Parameter: 0xB3})

	if got := len(oneDigit.Effect); got != 3 {
		t.Errorf("effect for command 0x03 is %q (%d columns), want 3",
			oneDigit.Effect, got)
	}

	if got := len(twoDigit.Effect); got != 4 {
		t.Errorf("effect for command 0x11 is %q (%d columns), want 4",
			twoDigit.Effect, got)
	}

	// same columns on screen, or the cells to their right no longer line up
	one := m.CellSegments(oneDigit, nil, false)
	two := m.CellSegments(twoDigit, nil, false)

	if a, b := utils.SegmentWidth(one), utils.SegmentWidth(two); a != b {
		t.Errorf("cells are %d and %d columns; a four column effect shifts the row", a, b)
	}

	if got := utils.SegmentWidth(one); got != tui.ChannelColumnWidth {
		t.Errorf("cell is %d columns, want %d", got, tui.ChannelColumnWidth)
	}

	// cut to the width, so a longer value cannot desync the scroll either
	long := m.CellSegments(tui.CellFields{
		Note: "C-4", Instrument: "01", Volume: "40", Effect: "12345",
	}, nil, false)

	if got := utils.SegmentWidth(long); got != tui.ChannelColumnWidth {
		t.Errorf("a 5 column effect rendered as %d columns, want %d",
			got, tui.ChannelColumnWidth)
	}
}

// TestEveryEffectWidthIsHeld: the field has to be wide enough for every effect a
// module can hold, not just the common ones.
func TestEveryEffectWidthIsHeld(t *testing.T) {
	for _, module := range utils.DemoModules(t) {
		t.Run(filepath.Base(module), func(t *testing.T) {
			m := utils.LoadModule(t, module)

			for _, line := range m.Tabs[tui.TabPattern].Formatted {
				for ch, cell := range line {
					if got := len(cell.Effect); got > tui.EffectFieldWidth {
						t.Errorf("channel %d: effect %q is %d columns, want at most %d",
							ch+1, cell.Effect, got, tui.EffectFieldWidth)
					}
				}
			}
		})
	}
}

// TestRowsStayAlignedAcrossThePattern is the end-to-end version of the same
// check (every row of a real pattern is the same width, whatever effects it
// contains)
func TestRowsStayAlignedAcrossThePattern(t *testing.T) {
	for _, module := range utils.DemoModules(t) {
		t.Run(filepath.Base(module), func(t *testing.T) {
			m := utils.LoadModule(t, module)
			utils.Resize(&m, 200, 50)

			view := &m.Tabs[tui.TabPattern].Pattern

			for offset := 0; offset <= view.MaxColOffset(m.Info.Channels); offset++ {
				view.ColOffset = offset
				view.Clamp(m.Info.Channels)

				for r := view.RowOffset; r < min(view.RowOffset+view.ViewRows(), view.Rows); r++ {
					if got := lipgloss.Width(m.PatternRow(r, *view)); got != 200 {
						t.Fatalf("offset %d row %d is %d columns, want 200", offset, r, got)
					}
				}
			}
		})
	}
}

// TestRowLabelsAreRowNumbers tests which pattern row the band is on, based on the
// gutter
func TestRowLabelsAreRowNumbers(t *testing.T) {
	m := utils.NewTestModel(120, 30)
	utils.LoadTestPattern(&m, 64)

	_, rows := utils.GridLines(m)

	view := m.Tabs[tui.TabPattern].Pattern

	for i, row := range rows {
		// the gutter is a segment, so the label is looked for in the
		// escapes-stripped text
		want := fmt.Sprintf("%-*d", tui.RowLabelNumberWidth, view.RowOffset+i)

		if !strings.Contains(utils.StripANSI(row), want) {
			t.Errorf("row line %d does not carry the label %q: %q", i, want, row)
		}
	}
}
