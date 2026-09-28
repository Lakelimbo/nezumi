package tui

import "fmt"

func (v visualizationID) String() string {
	switch v {
	case visualizationWaveform:
		return "waveform"
	case visualizationSpectrum:
		return "spectrum"
	case visualizationBars:
		return "bars"
	}

	return "unknown"
}

func (m Model) visualizationContent() []string {
	w, h := m.bodySize()
	if w == 0 || h == 0 {
		return []string{""}
	}

	return []string{
		fmt.Sprintf("visualizer: %s", m.Tabs[TabVisualization].Visualizer),
		"",
		fmt.Sprintf("canvas:   %d x %d", w, h),
		fmt.Sprintf("samples:  %d", len(m.AudioState.Wave)),
		fmt.Sprintf("rms:      %.4f", m.AudioState.RMS),
		"",
		"to-do: waveform, spectrum, bars",
	}
}
