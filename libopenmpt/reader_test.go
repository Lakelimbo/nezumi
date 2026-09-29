package libopenmpt_test

import (
	"io"
	"testing"

	"github.com/Lakelimbo/nezumi/libopenmpt"
)

func TestModuleReaderSeekRewindsModule(t *testing.T) {
	mod := openDemo(t, "../demo/necros-realization.s3m")
	reader := newTestReader(mod)

	buf := make([]byte, 64*1024)

	for range 32 {
		n, err := reader.Read(buf)
		if n == 0 || (err != nil && err != io.EOF) {
			t.Fatalf("Read: (%d, %v)", n, err)
		}
	}

	if order, row := mod.CurrentOrder(), mod.CurrentRow(); order == 0 && row == 0 {
		t.Fatal("reading did not advance the module")
	}

	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("Seek: %v", err)
	}

	if order, row := mod.CurrentOrder(), mod.CurrentRow(); order != 0 || row != 0 {
		t.Errorf("module is at %d:%d after seek, want 0:0", order, row)
	}

	if reader.Ended.Load() {
		t.Error("seek left the reader marked as finished")
	}

	if n, err := reader.Read(buf); n == 0 || err != nil {
		t.Errorf("Read returned (%d, %v) after seek, want audio", n, err)
	}
}

func TestModuleReaderSeekRejection(t *testing.T) {
	reader := newTestReader(openDemo(t, "../demo/necros-realization.s3m"))

	for _, tc := range []struct {
		offset int64
		whence int
	}{
		{1, io.SeekStart},
		{4096, io.SeekStart},
		{0, io.SeekEnd},
	} {
		if _, err := reader.Seek(tc.offset, tc.whence); err == nil {
			t.Errorf("Seek(%d, %d) succeeded, but expected an error", tc.offset, tc.whence)
		}
	}
}

func openDemo(t *testing.T, path string) *libopenmpt.Module {
	t.Helper()

	mod, err := libopenmpt.Open(path)
	if err != nil {
		t.Skipf("cannot open %s: %v", path, err)
	}

	t.Cleanup(mod.Close)
	return mod
}

func newTestReader(mod *libopenmpt.Module) *libopenmpt.ModuleReader {
	return &libopenmpt.ModuleReader{
		Module:     mod,
		SampleRate: 4800,
		Chunk:      make([]float32, 8192*2),
	}
}
