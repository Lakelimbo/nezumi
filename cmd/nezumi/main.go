package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/Lakelimbo/nezumi/internal/tui"
	"github.com/Lakelimbo/nezumi/libopenmpt"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:                  "Nezumi Music Tracker",
		Usage:                 "experimental TUI music (keygen) tracker",
		Commands:              []*cli.Command{},
		EnableShellCompletion: true,
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:     "module",
				Required: true,
			},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "no-tui",
				Usage: "plays the module without the TUI",
				Value: false,
			},
			&cli.IntFlag{
				Name:    "sample-rate",
				Aliases: []string{"sr"},
				Usage:   "sets the sample rate for libopenmpt",
				Value:   48000,
			},
			&cli.IntFlag{
				Name:    "audio-queue-depth",
				Aliases: []string{"aqd"},
				Usage:   "how many PCM Frames may sit between the audio callback and the TUI",
				Value:   4,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			file := c.StringArg("module")

			sampleRate := c.Int("sample-rate")
			audioQueueDepth := c.Int("audio-queue-depth")

			mod, err := libopenmpt.Open(file)
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

			player.Play()

			if c.Bool("no-tui") {
				fmt.Printf("Playing %s...", mod.Title())

				select {
				case <-player.Done():
					if err := player.Err(); err != nil {
						return err
					}
					return nil

				case <-ctx.Done():
					player.Stop()
					return ctx.Err()
				}
			}

			program := tea.NewProgram(tui.New(mod, player, audio), tea.WithFPS(30))
			if _, err := program.Run(); err != nil && !errors.Is(err, tea.ErrInterrupted) {
				log.Fatal(err)
			}

			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
