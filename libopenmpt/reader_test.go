package libopenmpt_test

import (
	"io"
	"testing"

	"github.com/Lakelimbo/nezumi/internal/utils"
	"github.com/Lakelimbo/nezumi/libopenmpt"
)

func TestModuleReaderSeekRequestedOrderRow(t *testing.T) {
	mod := openDemo(t, utils.DemoRealization)
	reader := newTestReader(mod)

	const (
		order = 14
		row   = 16
	)

	if _, err := reader.Seek(testOrderOffset(order, row), io.SeekStart); err != nil {
		t.Fatalf("Seek: %v", err)
	}

	if got := mod.CurrentOrder(); got != order {
		t.Errorf("row is %d after the seek, want %d", got, row)
	}

	// order landed on a different pattern, so the reader has to have dropped
	// the rendered chunk for the old one
	if n, err := reader.Read(make([]byte, 64*1024)); n == 0 || err != nil {
		t.Errorf("Read returned (%d, %v) after the seek, but want audio", n, err)
	}
}

func TestModuleReaderSeekStart(t *testing.T) {
	mod := openDemo(t, utils.DemoRealization)
	reader := newTestReader(mod)

	readChunks(t, reader, 32)

	if _, err := reader.Seek(testOrderOffset(9, 32), io.SeekStart); err != nil {
		t.Fatalf("Seek: %v", err)
	}

	// rewind depends on offset 0 == order 0 && row 0
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("Seek(0): %v", err)
	}

	if order, row := mod.CurrentOrder(), mod.CurrentRow(); order != 0 || row != 0 {
		t.Errorf("module ios at %d:%d after seeking to 0, want 0:0", order, row)
	}
}

func TestModuleReaderSeekClearFinishFlag(t *testing.T) {
	mod := openDemo(t, utils.DemoRealization)
	reader := newTestReader(mod)

	reader.Ended.Store(true)

	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("Seek: %v", err)
	}

	if reader.Ended.Load() {
		t.Error("seek left reader marked as finished, so playback could not continue")
	}
}

func TestModuleReaderSeekRejection(t *testing.T) {
	mod := openDemo(t, utils.DemoRealization)

	for _, tc := range []struct {
		offset int64
		whence int
	}{
		{-1, io.SeekStart},
		{0, io.SeekEnd},
		{0, io.SeekCurrent},
		{testOrderOffset(mod.NumOrders(), 0), io.SeekStart},
		{testOrderOffset(3, 4096), io.SeekStart},
	} {
		reader := newTestReader(mod)

		if _, err := reader.Seek(tc.offset, tc.whence); err == nil {
			t.Errorf("Seek(%d, %d) succeeded, but expected an error", tc.offset, tc.whence)
		}

		if got := mod.CurrentOrder(); got != 0 {
			t.Fatalf("a refused seek moved the module to order %d", got)
		}
	}
}

func openDemo(t *testing.T, song utils.DemoSong) *libopenmpt.Module {
	t.Helper()

	mod, err := libopenmpt.Open(string(song))
	if err != nil {
		t.Skipf("cannot open %s: %v", song, err)
	}

	t.Cleanup(mod.Close)
	return mod
}

func readChunks(t *testing.T, reader *libopenmpt.ModuleReader, n int) {
	t.Helper()

	buf := make([]byte, 64*1024)
	for range n {
		read, err := reader.Read(buf)
		if read == 0 || (err != nil && err != io.EOF) {
			t.Fatalf("Read: (%d, %v)", read, err)
		}
	}

	if order, row := reader.Module.CurrentOrder(), reader.Module.CurrentRow(); order == 0 && row == 0 {
		t.Fatal("reading did not advance the module, so the seek provides nothing")
	}
}

func newTestReader(mod *libopenmpt.Module) *libopenmpt.ModuleReader {
	return &libopenmpt.ModuleReader{
		Module:     mod,
		SampleRate: 4800,
		Chunk:      make([]float32, 8192*2),
	}
}

func testOrderOffset(order, row int) int64 {
	return int64(order)*65536 + int64(row)
}
