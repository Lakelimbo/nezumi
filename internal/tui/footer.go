package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

const FooterHeight = 1

func (m Model) FooterLine() string {
	switch {
	case m.CommandActive:
		return m.CommandLine()

	case len(m.PendingKeys) != 0:
		return m.WhichKeyLine()

	default:
		return m.StatusBar()
	}
}

func (m Model) footerBackground(content string) string {
	if m.Width <= 0 {
		return ""
	}

	return lipgloss.NewStyle().
		Background(m.Theme.StatusBg).
		Width(m.Width).
		Height(FooterHeight).
		MaxWidth(m.Width).
		MaxHeight(FooterHeight).
		Render(content)
}

func (m Model) StatusBar() string {
	if m.Width <= 0 {
		return ""
	}

	block := lipgloss.NewStyle().Bold(true).Padding(0, 1)

	marker := lipgloss.NewStyle().
		Background(m.Theme.Accent).
		Width(1).
		Height(FooterHeight).
		Render("")

	title := block.
		Background(m.Theme.StatusBg).
		Foreground(m.Theme.Title).
		MaxWidth(30).
		Render(orDash(m.Info.Title))

	format := block.
		Background(m.Theme.StatusBg).
		Foreground(m.Theme.Empty).
		MaxWidth(18).
		Render(orDash(m.Info.TypeLong))

	channels := block.
		Background(m.Theme.StatusBg).
		Foreground(m.Theme.RowNumber).
		Render(m.ChannelReadout())

	playback := block.
		Background(m.Theme.StatusBg).
		Padding(0, 2).
		Render(playbackStatus(m.Theme, m.Playing))

	position := block.
		Background(m.Theme.StatusBg).
		Foreground(m.Theme.Accent).
		Render(fmt.Sprintf("%d:%02d", m.Position.Pattern, m.Position.Row))

	used := lipgloss.Width(marker) +
		lipgloss.Width(title) +
		lipgloss.Width(format) +
		lipgloss.Width(channels) +
		lipgloss.Width(playback) +
		lipgloss.Width(position)

	spacer := lipgloss.NewStyle().
		Background(m.Theme.StatusBg).
		Render(strings.Repeat(" ", max(0, m.Width-used)))

	bar := lipgloss.JoinHorizontal(lipgloss.Top,
		marker,
		title,
		spacer,
		format,
		channels,
		playback,
		position,
	)

	return m.footerBackground(bar)
}

func (m Model) ChannelReadout() string {
	total := max(1, m.Info.Channels)
	view := m.Tabs[TabPattern].Pattern

	if view.MaxColOffset(total) == 0 {
		return ""
	}

	first, last := VisibleChannels(view.ColOffset, view.Avail(), total)

	// MaxColOffset is only non-zero when the grid does not fit, so at least
	// one channel is always off screen here and the readout always has
	// something to say.
	return fmt.Sprintf("ch %d-%d/%d", first+1, min(last, total-1)+1, total)
}

func (m Model) CommandLine() string {
	prompt := lipgloss.NewStyle().Foreground(m.Theme.Accent).Render(":")

	return m.footerBackground(prompt + m.Command.View())
}

func (m Model) WhichKeyLine() string {
	prefix := lipgloss.NewStyle().
		Foreground(m.Theme.Accent).
		Render(strings.Join(m.PendingKeys, " "))

	hint := lipgloss.NewStyle().
		Foreground(m.Theme.RowNumber).
		Render(" " + leaderKeyList())

	return m.footerBackground(prefix + hint)
}

func playbackStatus(th Theme, isPlaying bool) string {
	if isPlaying {
		return lipgloss.NewStyle().
			Foreground(th.Accent).
			Render(string(IconPlay) + " Playing")
	}

	return lipgloss.NewStyle().
		Foreground(th.Empty).
		Render(string(IconPause) + " Paused")
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}

	return s
}
