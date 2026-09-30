package tui

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	NoteFieldWidth      = 4 // "C-4 "
	ValueFieldWidth     = 3 // "01 "
	EffectFieldWidth    = 4 // "12FA"
	CellWidth           = NoteFieldWidth + 2*ValueFieldWidth + EffectFieldWidth
	TailWidth           = 2 // group separator slot
	ChannelColumnWidth  = CellWidth + TailWidth
	RowNumberWidth      = 5 // "  0 |"
	RowLabelNumberWidth = 4 // "  0 "
	ChannelGroupSize    = 4
)

const PatternHeaderHeight = 1

type Segment struct {
	Text string
	fg   color.Color
	bg   color.Color
}

func (s Segment) Render() string {
	return seg(s.Text, s.fg, s.bg)
}

func field(plain string, width int, fg, bg color.Color) Segment {
	return Segment{
		Text: fmt.Sprintf("%-*s", width, CutColumns(plain, 0, width)),
		fg:   fg,
		bg:   bg,
	}
}

func renderSegments(segs []Segment) string {
	var b strings.Builder

	for _, s := range segs {
		b.WriteString(s.Render())
	}

	return b.String()
}

func sliceSegments(segs []Segment, from, width int) []Segment {
	if width <= 0 {
		return nil
	}

	end := from + width
	out := make([]Segment, 0, len(segs))
	col := 0

	for _, s := range segs {
		w := lipgloss.Width(s.Text)

		switch next := col + w; {
		case w == 0: // nothing to show, and nothing to advance past
			continue

		case next <= from: // entirely left of the window
			col = next
			continue

		case col >= end: // entirely right of it, and so is the rest
			return out
		}

		// overlap expressed relative to this segment
		lo, hi := max(from-col, 0), min(end-col, w)
		out = append(out, Segment{Text: CutColumns(s.Text, lo, hi), fg: s.fg, bg: s.bg})

		col += w
	}

	return out
}

func CutColumns(text string, lo, hi int) string {
	if hi <= lo {
		return ""
	}

	runes := []rune(text)

	return string(runes[min(lo, len(runes)):min(hi, len(runes))])
}

func clampInt(v, low, high int) int {
	return min(max(v, low), max(low, high))
}

type PatternView struct {
	Rows      int
	Row       int
	RowOffset int
	ColOffset int
	Width     int
	Height    int
}

func (v PatternView) ViewRows() int {
	return max(0, v.Height-PatternHeaderHeight)
}

func (v PatternView) Avail() int {
	return max(0, v.Width-RowNumberWidth)
}

func (v PatternView) MaxColOffset(channels int) int {
	return max(0, max(1, channels)*ChannelColumnWidth-v.Avail())
}

func VisibleChannels(colOffset, avail, channels int) (first, last int) {
	channels = max(1, channels)

	first = min(colOffset/ChannelColumnWidth, channels-1)
	last = min(max((colOffset+avail-1)/ChannelColumnWidth, first), channels-1)

	return first, last
}

func (v *PatternView) Clamp(channels int) {
	v.Row = clampInt(v.Row, 0, max(0, v.Rows-1))

	rows := v.ViewRows()
	last := max(0, v.Rows-rows)

	v.RowOffset = clampInt(v.RowOffset, 0, last)

	// the selection is what follow mode tracks, so it wins over the scroll
	// position: move the window onto it rather than the other way round
	if rows > 0 {
		if v.Row < v.RowOffset {
			v.RowOffset = v.Row
		} else if v.Row >= v.RowOffset+rows {
			v.RowOffset = v.Row - rows + 1
		}

		v.RowOffset = clampInt(v.RowOffset, 0, last)
	}

	v.ColOffset = clampInt(v.ColOffset, 0, v.MaxColOffset(channels))
}

func (v *PatternView) MoveRow(delta, channels int) {
	v.Row += delta
	v.Clamp(channels)
}

func (v *PatternView) GotoTop(channels int) {
	v.Row = 0
	v.Clamp(channels)
}

func (v *PatternView) GotoBottom(channels int) {
	v.Row = v.Rows // one past the end. Clamp pulls it back onto the last row
	v.Clamp(channels)
}

func (v *PatternView) CentreOn(row, channels int) {
	v.Row = row

	if rows := v.ViewRows(); rows > 0 {
		v.RowOffset = row - rows/2
	}

	v.Clamp(channels)
}

func (v *PatternView) ScrollCols(delta, channels int) {
	v.ColOffset += delta
	v.Clamp(channels)
}

func (m *Model) FollowPatternRow(row int) {
	tab := &m.Tabs[TabPattern]
	if !tab.Follow || len(tab.Formatted) == 0 {
		return
	}

	tab.Pattern.CentreOn(row, m.Info.Channels)
}

func (m Model) CursorIsTransport() bool {
	return !m.Playing
}

func (m Model) PatternGrid() string {
	v := m.Tabs[TabPattern].Pattern

	// below the gutter there is no grid to draw, and a lone channel header
	// with no rows under it is just noise
	if v.Avail() <= 0 || v.ViewRows() <= 0 {
		return ""
	}

	lines := make([]string, 0, v.ViewRows()+1)
	lines = append(lines, m.channelHeader(v))

	for r := v.RowOffset; r < min(v.RowOffset+v.ViewRows(), v.Rows); r++ {
		lines = append(lines, m.PatternRow(r, v))
	}

	return strings.Join(lines, "\n")
}

func (m Model) channelHeader(v PatternView) string {
	channels := max(1, m.Info.Channels)

	var grid strings.Builder
	for ch := range channels {
		fmt.Fprintf(&grid, "%-*s", ChannelColumnWidth, " Ch "+strconv.Itoa(ch+1))
	}

	plain := fmt.Sprintf("%-*s", RowNumberWidth, "#") +
		CutColumns(grid.String(), v.ColOffset, v.ColOffset+v.Avail())

	return lipgloss.NewStyle().
		Bold(true).
		Foreground(m.Theme.Header).
		Render(padTo(plain, v.Width))
}

func (m Model) PatternRow(r int, v PatternView) string {
	th := m.Theme

	var bg color.Color
	if r == v.Row {
		bg = th.Highlight
	}

	gutter := renderSegments([]Segment{
		field(strconv.Itoa(r), RowLabelNumberWidth, th.RowNumber, bg),
		field(groupEdge, 1, th.Separator, bg),
	})

	body := m.channelArea(m.Tabs[TabPattern].Formatted[r], v, bg)

	if fill := v.Width - RowNumberWidth - lipgloss.Width(body); fill > 0 {
		body += seg(strings.Repeat(" ", fill), nil, bg)
	}

	return gutter + body
}

func (m Model) channelArea(line []CellFields, v PatternView, bg color.Color) string {
	first, last := VisibleChannels(v.ColOffset, v.Avail(), max(1, m.Info.Channels))
	last = min(last, len(line)-1)

	if last < first {
		return ""
	}

	segs := m.ChannelSegments(line[first:last+1], first, bg)

	return renderSegments(sliceSegments(
		segs,
		v.ColOffset-first*ChannelColumnWidth,
		v.Avail(),
	))
}

func (m Model) ChannelSegments(line []CellFields, first int, bg color.Color) []Segment {
	segs := make([]Segment, 0, len(line)*5)

	for i, c := range line {
		ch := first + i
		segs = append(segs, m.CellSegments(c, bg, ch > 0 && ch%ChannelGroupSize == 0)...)
	}

	return segs
}

func (m Model) CellSegments(c CellFields, bg color.Color, boundary bool) []Segment {
	th := m.Theme

	note, inst, vol, eff := th.Note, th.Instrument, th.Volume, th.Effect

	switch {
	case c.empty():
		note, inst, vol, eff = th.Empty, th.Empty, th.Empty, th.Empty

	default:
		switch c.Note {
		case emptyNote:
			note = th.Empty
		case noteOffStr, noteCutStr, unknownNote:
			note = th.NoteOff
		}

		if c.Instrument == emptyNumber {
			inst = th.Empty
		}

		if c.Volume == emptyNumber {
			vol = th.Empty
		}
	}

	segs := []Segment{
		field(c.Note, NoteFieldWidth, note, bg),
		field(c.Instrument, ValueFieldWidth, inst, bg),
		field(c.Volume, ValueFieldWidth, vol, bg),
		field(c.Effect, EffectFieldWidth, eff, bg),
	}

	if boundary {
		return append(segs,
			field(groupEdge, 1, th.Separator, bg),
			field(" ", 1, nil, bg),
		)
	}

	return append(segs, field("  ", TailWidth, nil, bg))
}

func padTo(plain string, width int) string {
	if fill := width - lipgloss.Width(plain); fill > 0 {
		return plain + strings.Repeat(" ", fill)
	}

	return plain
}
