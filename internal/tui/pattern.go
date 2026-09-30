package tui

import (
	"fmt"

	"github.com/Lakelimbo/nezumi/libopenmpt"
)

var noteNames = [12]string{
	"C-", "C#", "D-", "D#", "E-", "F-",
	"F#", "G-", "G#", "A-", "A#", "B-",
}

const (
	emptyNote    = "..."
	emptyNumber  = ".."
	noteOffStr   = "== "
	noteCutStr   = "^^ "
	unknownNote  = "?? "
	groupEdge    = "|"
	maxNoteValue = 120
)

type CellFields struct {
	Note       string
	Instrument string
	Volume     string
	Effect     string
}

func (c CellFields) empty() bool {
	return c.Note == emptyNote &&
		c.Instrument == emptyNumber &&
		c.Volume == emptyNumber &&
		c.Effect == emptyNumber
}

func noteName(note uint8) string {
	switch note {
	case 0:
		return emptyNote
	case 254:
		return noteOffStr
	case 255:
		return noteCutStr
	}

	if note > maxNoteValue {
		return unknownNote
	}

	n := int(note) - 1
	return fmt.Sprintf("%s%d", noteNames[n%12], n/12)
}

func ParseCell(c libopenmpt.Cell) CellFields {
	inst, vol, eff := emptyNumber, emptyNumber, emptyNumber

	if c.Instrument != 0 {
		inst = fmt.Sprintf("%02X", c.Instrument)
	}

	if c.Volume != 0 {
		vol = fmt.Sprintf("%02X", c.Volume)
	}

	// libopenmpt reports the command with a 1-based index.
	//
	// Also, decided to format (at least for now) as a hex digit rather than
	// letters from G and above because they are dependent on the format and
	// I need to learn how, say, OpenMPT does it specificaly
	if c.Effect != 0 {
		eff = fmt.Sprintf("%X%02X", c.Effect-1, c.Parameter)
	}

	return CellFields{
		Note:       noteName(c.Note),
		Instrument: inst,
		Volume:     vol,
		Effect:     eff,
	}
}

func (m *Model) LoadPattern(data libopenmpt.PatternData) {
	m.Pattern = data

	tab := &m.Tabs[TabPattern]
	tab.Formatted = m.formatPattern()
	tab.Pattern.Rows = len(tab.Formatted)
	tab.Pattern.Clamp(m.Info.Channels)
}

func (m Model) formatPattern() [][]CellFields {
	if m.Pattern.Rows == 0 || len(m.Pattern.Cells) == 0 {
		return nil
	}

	grid := make([][]CellFields, m.Pattern.Rows)
	for row, cells := range m.Pattern.Cells {
		line := make([]CellFields, len(cells))
		for ch, cell := range cells {
			line[ch] = ParseCell(cell)
		}

		grid[row] = line
	}

	return grid
}
