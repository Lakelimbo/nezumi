package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/Lakelimbo/nezumi/libopenmpt"
)

const positionPollInterval = 50 * time.Millisecond

func pollPosition(mod *libopenmpt.Module) tea.Cmd {
	return func() tea.Msg {
		// wait first so the mutex is held for the read and nothing else.
		<-time.After(positionPollInterval)

		return positionMsg{Position: mod.Position()}
	}
}

func loadPattern(mod *libopenmpt.Module, index int) tea.Cmd {
	return func() tea.Msg {
		data, err := mod.ReadPattern(index)

		return PatternMsg{
			Index: index,
			Data:  data,
			Err:   err,
		}
	}
}

func waitForAudio(audio <-chan libopenmpt.PCMFrame, done <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		select {
		case frame := <-audio:
			return audioMsg{frame}
		case <-done:
			return audioStoppedMsg{}
		}
	}
}

func waitForPlaybackDone(done <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		<-done

		return playbackDoneMsg{}
	}
}

func leaderTimeout(token uint64) tea.Cmd {
	return tea.Tick(
		leaderWait,
		func(time.Time) tea.Msg {
			return leaderExpiredMsg{Token: token}
		},
	)
}

func rearmPlayback(audio <-chan libopenmpt.PCMFrame, player *libopenmpt.Player) tea.Cmd {
	return tea.Batch(
		waitForAudio(audio, player.Done()),
		waitForPlaybackDone(player.Done()),
	)
}
