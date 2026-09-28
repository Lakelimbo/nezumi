package utils

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Lakelimbo/nezumi/internal/tui"
)

func GridLines(m tui.Model) (header string, rows []string) {
	lines := strings.Split(m.PatternGrid(), "\n")

	return lines[0], lines[1:]
}

func StripANSI(line string) string {
	return ansi.Strip(line)
}

func SegmentWidth(segs []tui.Segment) int {
	total := 0

	for _, s := range segs {
		total += lipgloss.Width(s.Text)
	}

	return total
}

func BackgroundSGR(c color.Color) string {
	r, g, b, _ := c.RGBA()

	return fmt.Sprintf("48;2;%d;%d;%d", r>>8, g>>8, b>>8)
}

func ForegroundSGR(c color.Color) string {
	r, g, b, _ := c.RGBA()

	return fmt.Sprintf("38;2;%d;%d;%d", r>>8, g>>8, b>>8)
}
