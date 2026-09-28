package tui

import (
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

// infoLabelWidth is the fixed width of the label column, chosen so the longest
// label ("duration") plus a gap still leaves room for a value
const infoLabelWidth = 10

func (m Model) infoContent() []string {
	lines := []string{
		m.infoHeading("module"),
		m.infoRow("title", m.Info.Title),
		m.infoRow("artist", m.Info.Artist),
		m.infoRow("tracker", m.Info.Tracker),
		m.infoRow("format", m.Info.TypeLong),
		m.infoRow("duration", m.infoDuration()),
		m.infoRow("channels", strconv.Itoa(m.Info.Channels)),
		m.infoRow("patterns", strconv.Itoa(m.Info.Patterns)),
		"",
		m.infoHeading("playback"),
		m.infoRow("state", playbackStatus(m.Theme, m.Playing)),
		m.infoRow("order", strconv.Itoa(m.Position.Order)),
		m.infoRow("pattern", strconv.Itoa(m.Position.Pattern)),
		m.infoRow("row", strconv.Itoa(m.Position.Row)),
	}

	if m.Status != "" {
		lines = append(lines,
			"",
			m.infoHeading("status"),
			lipgloss.NewStyle().Foreground(m.Theme.NoteOff).Render(m.Status),
		)
	}

	return lines
}

func (m Model) infoHeading(text string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(m.Theme.Accent).
		Render(strings.ToUpper(text))
}

func (m Model) infoRow(label, value string) string {
	name := lipgloss.NewStyle().
		Foreground(m.Theme.RowNumber).
		Width(infoLabelWidth).
		Render(label)

	return name + orDash(value)
}

func (m Model) infoDuration() string {
	if m.Info.Duration <= 0 {
		return ""
	}

	return m.Info.Duration.Round(time.Second).String()
}
