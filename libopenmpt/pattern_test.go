package libopenmpt_test

import (
	"testing"

	"github.com/Lakelimbo/nezumi/libopenmpt"
)

type DemoSong string

const (
	Demo3DGalax       DemoSong = "../demo/dubmood-3d_galax.xm"
	DemoDreamstone    DemoSong = "../demo/nightbeat-dreamstone.it"
	DemoRealization   DemoSong = "../demo/necros-realization.s3m"
	DemoWorldOfDentro DemoSong = "../demo/4mat-world_of_dentro.mod"
)

func TestPattern(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		module      DemoSong
		numPatterns int
		numChannels int
	}{
		{
			module:      Demo3DGalax,
			numPatterns: 44,
			numChannels: 16,
		},
		{
			module:      DemoDreamstone,
			numPatterns: 38,
			numChannels: 18,
		},
		{
			module:      DemoRealization,
			numPatterns: 25,
			numChannels: 9,
		},
		{
			module:      DemoWorldOfDentro,
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
