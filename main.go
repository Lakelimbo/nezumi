package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/Lakelimbo/nezumi/internal/tui"
	"github.com/Lakelimbo/nezumi/libopenmpt"
)

const sampleRate = 48000

// audioQueueDepth is how many PCM frames may sit between the audio callback and
// the TUI. The callback drops frames rather than block, since blocking it would
// glitch playback; this only decides how much of a burst the TUI can catch.
const audioQueueDepth = 4

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s module.xm\n", os.Args[0])
		os.Exit(2)
	}

	mod, err := libopenmpt.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer mod.Close()

	player, err := libopenmpt.NewPlayer(mod, sampleRate, 0)
	if err != nil {
		log.Fatal(err)
	}
	defer player.Stop()

	audio := make(chan libopenmpt.PCMFrame, audioQueueDepth)
	player.SetPCMHandler(func(p libopenmpt.PCMFrame) {
		select {
		case audio <- p:
		default:
		}
	})

	program := tea.NewProgram(tui.New(mod, player, audio), tea.WithFPS(30))

	// Start audio only once the program is built, so a failure to open the
	// device happens before playback has already begun.
	player.Play()

	if _, err := program.Run(); err != nil && !errors.Is(err, tea.ErrInterrupted) {
		log.Fatal(err)
	}
}
