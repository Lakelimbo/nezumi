package libopenmpt_test

import (
	"testing"

	"github.com/Lakelimbo/nezumi/internal/utils"
	"github.com/Lakelimbo/nezumi/libopenmpt"
)

func TestPattern(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		module      utils.DemoSong
		numPatterns int
		numChannels int
	}{
		{
			module:      utils.Demo3DGalax,
			numPatterns: 44,
			numChannels: 16,
		},
		{
			module:      utils.DemoDreamstone,
			numPatterns: 38,
			numChannels: 18,
		},
		{
			module:      utils.DemoRealization,
			numPatterns: 25,
			numChannels: 9,
		},
		{
			module:      utils.DemoWorldOfDentro,
			numPatterns: 8,
			numChannels: 4,
		},
	}

	for _, s := range scenarios {
		t.Run(string(s.module), func(t *testing.T) {
			mod, _ := libopenmpt.Open(string(s.module))
			defer mod.Close()

			if mod.NumPatterns() != s.numPatterns {
				t.Fatalf("expected %d patterns, got %d", s.numPatterns, mod.NumPatterns())
			}
			if mod.NumChannels() != s.numChannels {
				t.Fatalf("expected %d channels, got %d", s.numChannels, mod.NumChannels())
			}
		})
	}
}
