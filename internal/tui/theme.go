package tui

import (
	"fmt"
	"image/color"

	"charm.land/lipgloss/v2"
)

type Theme struct {
	Note       color.Color
	NoteOff    color.Color
	Instrument color.Color
	Volume     color.Color
	Effect     color.Color

	RowNumber color.Color
	Empty     color.Color
	Separator color.Color
	Header    color.Color
	Highlight color.Color

	StatusBg color.Color
	Accent   color.Color
	Title    color.Color
}

// TO-DO:
// make a better default theme
func DefaultTheme() Theme {
	return Theme{
		Note:       lipgloss.Color("#7cd4ff"),
		NoteOff:    lipgloss.Color("#f7768e"),
		Instrument: lipgloss.Color("#e0af68"),
		Volume:     lipgloss.Color("#9ece6a"),
		Effect:     lipgloss.Color("#bb9af7"),

		RowNumber: lipgloss.BrightBlack,
		Empty:     lipgloss.Color("#64616c"),
		Separator: lipgloss.Color("#414868"),
		Header:    lipgloss.Color("#a9b1d6"),
		Highlight: lipgloss.Color("#2f3549"),

		StatusBg: lipgloss.Color("#1a1b26"),
		Accent:   lipgloss.Color("#7aa2f7"),
		Title:    lipgloss.Color("#c0caf5"),
	}
}

func seg(plain string, fg, bg color.Color) string {
	return lipgloss.NewStyle().
		Foreground(fg).
		Background(bg).
		Render(plain)
}

func pad(plain string, width int, fg, bg color.Color) string {
	return seg(fmt.Sprintf("%-*s", width, plain), fg, bg)
}
