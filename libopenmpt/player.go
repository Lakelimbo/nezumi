package libopenmpt

import (
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/ebitengine/oto/v3"
)

// DefaultChunkFrames is the number of stereo frames rendered per chunk.
const DefaultChunkFrames = 4096

// Player plays a Module through the default audio device using oto
// (PulseAudio on Linux). Rendering is pull-driven: oto reads from the module
// whenever its buffer needs more audio, so playback is paced by the sound
// card, not by a timer.
//
// Only one Player (and therefore one oto context) may exist per process.
type Player struct {
	module     *Module
	sampleRate int

	ctx    *oto.Context
	player *oto.Player
	reader *moduleReader

	done      chan struct{}
	doneOnce  sync.Once
	startOnce sync.Once
	stopOnce  sync.Once
}

// NewPlayer creates a ready-to-use Player for mod at sampleRate and starts no
// audio until Play is called. chunkFrames sets how many stereo frames are
// rendered per read; use DefaultChunkFrames if you are unsure.
func NewPlayer(mod *Module, sampleRate int, chunkFrames int) (*Player, error) {
	if mod == nil {
		return nil, fmt.Errorf("libopenmpt: nil module")
	}
	if chunkFrames <= 0 {
		chunkFrames = DefaultChunkFrames
	}

	opts := &oto.NewContextOptions{
		SampleRate:      sampleRate,
		ChannelCount:    2, // libopenmpt renders interleaved stereo
		Format:          oto.FormatFloat32LE,
		ApplicationName: "nezumi",
	}

	ctx, ready, err := oto.NewContext(opts)
	if err != nil {
		return nil, fmt.Errorf("libopenmpt: create audio context: %w", err)
	}
	<-ready
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("libopenmpt: init audio context: %w", err)
	}

	reader := &moduleReader{
		module:     mod,
		sampleRate: sampleRate,
		chunk:      make([]float32, chunkFrames*2),
	}

	p := &Player{
		module:     mod,
		sampleRate: sampleRate,
		ctx:        ctx,
		reader:     reader,
		done:       make(chan struct{}),
	}
	p.player = ctx.NewPlayer(reader)

	return p, nil
}

// Play starts playback and returns immediately.
func (p *Player) Play() {
	p.startOnce.Do(func() {
		p.player.Play()
		go p.watchDone()
	})
}

// Pause pauses playback, keeping the position and buffered audio.
func (p *Player) Pause() {
	p.player.Pause()
}

// Resume continues playback after Pause.
func (p *Player) Resume() {
	p.player.Play()
}

// Volume returns the current volume (1 is the default).
func (p *Player) Volume() float64 {
	return p.player.Volume()
}

// SetVolume sets the volume; values above 1 amplify and may clip.
func (p *Player) SetVolume(volume float64) {
	p.player.SetVolume(volume)
}

// Frames returns the number of frames rendered so far.
func (p *Player) Frames() int64 {
	return p.reader.frames.Load()
}

// Done is closed when playback ends, either because the module finished or
// because Stop was called.
func (p *Player) Done() <-chan struct{} {
	return p.done
}

// Stop halts playback immediately and releases the audio resources. It is
// safe to call multiple times and from any goroutine.
func (p *Player) Stop() {
	p.stopOnce.Do(func() {
		p.reader.stop()
		// Stop pulling from the module and keep whatever is buffered from
		// reaching the speaker.
		p.player.PauseAndStopReading()
		p.closeDone()
	})
}

// Err returns the first error that occurred in the audio driver or while
// rendering, if any.
func (p *Player) Err() error {
	if err := p.player.Err(); err != nil {
		return err
	}
	return p.ctx.Err()
}

func (p *Player) closeDone() {
	p.doneOnce.Do(func() { close(p.done) })
}

// watchDone closes Done when the module reaches the end and the audio buffer
// has played out.
func (p *Player) watchDone() {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		if p.reader.stopped.Load() {
			p.closeDone()
			return
		}
		if p.reader.ended.Load() && !p.player.IsPlaying() {
			p.closeDone()
			return
		}
	}
}

type PCMFrame struct {
	SampleRate int
	Samples    []float32 // interleaved stereo
}

type PCMHandler func(PCMFrame)

func (p *Player) SetPCMHandler(handler PCMHandler) {
	p.reader.pcmHandler = handler
}

// moduleReader is an io.Reader that renders libopenmpt audio on demand. It is
// consumed by oto's internal player loop (a single goroutine), so the fields
// accessed there need only be guarded against Stop, which happens through
// atomics.
type moduleReader struct {
	module     *Module
	sampleRate int

	chunk     []float32
	chunkData []byte
	chunkPos  int
	chunkLen  int

	frames  atomic.Int64
	stopped atomic.Bool
	ended   atomic.Bool

	pcmHandler PCMHandler
}

// Read fills p with interleaved float32 stereo audio rendered from the
// module. It returns io.EOF once the module has finished.
func (r *moduleReader) Read(p []byte) (int, error) {
	if r.stopped.Load() {
		return 0, io.EOF
	}

	if r.chunkPos >= r.chunkLen {
		n, err := r.module.Render(r.sampleRate, r.chunk)
		if err != nil {
			return 0, err
		}
		if n == 0 {
			r.ended.Store(true)
			return 0, io.EOF
		}
		r.frames.Add(int64(n))
		r.chunkData = float32Bytes(r.chunk[:n*2])
		r.chunkLen = len(r.chunkData)
		r.chunkPos = 0

		if r.pcmHandler != nil {
			frame := PCMFrame{
				SampleRate: r.sampleRate,
				Samples:    append([]float32(nil), r.chunk[:n*2]...),
			}

			r.pcmHandler(frame)
		}
	}

	n := copy(p, r.chunkData[r.chunkPos:r.chunkLen])
	r.chunkPos += n
	return n, nil
}

func (r *moduleReader) stop() {
	r.stopped.Store(true)
}

// float32Bytes reinterprets a float32 slice as bytes without copying. The
// result is only valid while s is alive and unmodified.
func float32Bytes(s []float32) []byte {
	if len(s) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&s[0])), len(s)*4)
}
