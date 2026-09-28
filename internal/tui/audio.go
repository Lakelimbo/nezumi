package tui

import (
	"math"

	"github.com/Lakelimbo/nezumi/libopenmpt"
)

const maxWaveSamples = 4096

func (m *Model) ingestAudio(frame libopenmpt.PCMFrame) {
	mono := make([]float32, len(frame.Samples)/2)
	for i := range mono {
		mono[i] = (frame.Samples[i*2] + frame.Samples[i*2+1]) * 0.5
	}

	if len(m.AudioState.Wave) != maxWaveSamples {
		m.AudioState.Wave = make([]float32, maxWaveSamples)
	}

	if len(mono) == 0 {
		m.AudioState.RMS = 0
		return
	}

	// shift left by the new samples and append them, so the newest audio ends
	// up at the tail.
	//
	// The short and long frame cases are the same operation
	if len(mono) >= maxWaveSamples {
		copy(m.AudioState.Wave, mono[len(mono)-maxWaveSamples:])
	} else {
		copy(m.AudioState.Wave, m.AudioState.Wave[len(mono):])
	}

	copy(m.AudioState.Wave[maxWaveSamples-len(mono):], mono)

	var energy float64
	for _, sample := range mono {
		energy += float64(sample) * float64(sample)
	}

	m.AudioState.RMS = math.Sqrt(energy / float64(len(mono)))
}
