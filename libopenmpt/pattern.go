package libopenmpt

import "fmt"

// Pattern is a view over one pattern of the module. It keeps a reference to
// the owning Module, so you must not Close the module while a Pattern is in
// use.
type Pattern struct {
	module *Module
	index  int
	rows   int
}

type PatternData struct {
	Index int
	Rows  int
	Cells [][]Cell
}

func (p *Pattern) Index() int { return p.index }

func (p *Pattern) Rows() int { return p.rows }

// Cell holds the raw data of one row/channel cell. All fields use libopenmpt
// semantics: Note 0 means no note or note-off, and blank fields are 0. Use the
// Command constants below to interpret them if needed.
type Cell struct {
	Note       uint8
	Instrument uint8
	Effect     uint8
	Volume     uint8
	Parameter  uint8
}

// Libopenmpt command indices, matching libopenmpt.h.
const (
	CommandNote       = 0 // OPENMPT_MODULE_COMMAND_NOTE
	CommandInstrument = 1 // OPENMPT_MODULE_COMMAND_INSTRUMENT
	CommandEffect     = 3 // OPENMPT_MODULE_COMMAND_EFFECT
	CommandVolume     = 4 // OPENMPT_MODULE_COMMAND_VOLUME
	CommandParameter  = 5 // OPENMPT_MODULE_COMMAND_PARAMETER
)

// Cell reads one cell of the pattern. channel must be in [0, NumChannels).
func (p *Pattern) Cell(row, channel int) (Cell, error) {
	if row < 0 || row >= p.rows {
		return Cell{}, fmt.Errorf("row out of range: %d", row)
	}
	if channel < 0 || channel >= p.module.NumChannels() {
		return Cell{}, fmt.Errorf("channel out of range: %d", channel)
	}

	return Cell{
		Note:       p.module.patternCommand(p.index, row, channel, CommandNote),
		Instrument: p.module.patternCommand(p.index, row, channel, CommandInstrument),
		Effect:     p.module.patternCommand(p.index, row, channel, CommandEffect),
		Volume:     p.module.patternCommand(p.index, row, channel, CommandVolume),
		Parameter:  p.module.patternCommand(p.index, row, channel, CommandParameter),
	}, nil
}

// Command returns the raw value of a single libopenmpt pattern command (see
// the Command constants) at the given row and channel.
func (p *Pattern) Command(row, channel, command int) (uint8, error) {
	if row < 0 || row >= p.rows {
		return 0, fmt.Errorf("row out of range: %d", row)
	}
	if channel < 0 || channel >= p.module.NumChannels() {
		return 0, fmt.Errorf("channel out of range: %d", channel)
	}

	return p.module.patternCommand(p.index, row, channel, command), nil
}
