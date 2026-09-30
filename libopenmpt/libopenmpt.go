package libopenmpt

/*
#cgo pkg-config: libopenmpt

#include "libopenmpt.h"

#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

// Modterm wrapper around the libopenmpt module handle, so Go only ever
// deals with a single opaque pointer.
typedef struct {
  openmpt_module *module;
} modterm_module;

static void modterm_noop_log(const char *msg, void *user) {
  (void)msg;
  (void)user;
}

static int modterm_noop_error(int err, void *user) {
  (void)err;
  (void)user;
  return OPENMPT_ERROR_FUNC_RESULT_NONE;
}

static modterm_module *modterm_open_impl(
    const void *data, size_t size, int *err, const char **errmsg
) {
  openmpt_module *module = openmpt_module_create_from_memory2(
      data, size, modterm_noop_log, NULL, modterm_noop_error, NULL, err, errmsg,
      NULL
  );
  if (module == NULL) {
    return NULL;
  }

  modterm_module *wrapper = (modterm_module *)malloc(sizeof(modterm_module));
  if (wrapper == NULL) {
    openmpt_module_destroy(module);
    return NULL;
  }
  wrapper->module = module;
  return wrapper;
}

static void modterm_close_impl(modterm_module *wrapper) {
  if (wrapper == NULL) {
    return;
  }
  if (wrapper->module != NULL) {
    openmpt_module_destroy(wrapper->module);
  }
  free(wrapper);
}

static size_t modterm_render_impl(
    modterm_module *wrapper, int32_t samplerate, size_t frames, float *output
) {
  return openmpt_module_read_interleaved_float_stereo(
      wrapper->module, samplerate, frames, output
  );
}

static const char *modterm_metadata_impl(modterm_module *wrapper,
                                         const char *key) {
  return openmpt_module_get_metadata(wrapper->module, key);
}

static double modterm_duration_impl(modterm_module *wrapper) {
  return openmpt_module_get_duration_seconds(wrapper->module);
}

static int32_t modterm_currentorder_impl(modterm_module *wrapper) {
  return openmpt_module_get_current_order(wrapper->module);
}

static int32_t modterm_currentpattern_impl(modterm_module *wrapper) {
  return openmpt_module_get_current_pattern(wrapper->module);
}

static int32_t modterm_currentrow_impl(modterm_module *wrapper) {
  return openmpt_module_get_current_row(wrapper->module);
}

static int modterm_setpositionorderrow_impl(modterm_module *wrapper,
                                            int32_t order, int32_t row) {
  return openmpt_module_set_position_order_row(wrapper->module, order, row) >=
         0.0;
}

static int32_t modterm_numchannels_impl(modterm_module *wrapper) {
  return openmpt_module_get_num_channels(wrapper->module);
}

static int32_t modterm_numpatterns_impl(modterm_module *wrapper) {
  return openmpt_module_get_num_patterns(wrapper->module);
}

static int32_t modterm_numorders_impl(modterm_module *wrapper) {
	return openmpt_module_get_num_orders(wrapper->module);
}

static int32_t modterm_orderpattern_impl(modterm_module *wrapper,
																				 int32_t order) {
	return openmpt_module_get_order_pattern(wrapper->module, order);
}

static int modterm_orderplayable_impl(modterm_module *wrapper,
																		 int32_t order) {
	return openmpt_module_is_order_skip_entry(wrapper->module, order) == 0 &&
				 openmpt_module_is_order_stop_entry(wrapper->module, order) == 0;
}

static int32_t modterm_patternrows_impl(modterm_module *wrapper,
                                        int32_t pattern) {
  return openmpt_module_get_pattern_num_rows(wrapper->module, pattern);
}

static uint8_t modterm_patterncommand_impl(modterm_module *wrapper,
                                           int32_t pattern, int32_t row,
                                           int32_t channel, int command) {
  return openmpt_module_get_pattern_row_channel_command(
      wrapper->module, pattern, row, channel, command
  );
}
*/
import "C"

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"
	"unsafe"
)

// Module wraps an openmpt_module handle. It is safe to use from a single
// goroutine at a time; libopenmpt is not thread-safe.
type Module struct {
	ptr *C.modterm_module
	mu  sync.Mutex
}

// Open loads a tracker module from a file. The data is copied into libopenmpt
// during creation, so the file is not kept open afterwards.
func Open(path string) (*Module, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read module: %w", err)
	}

	var dataPtr unsafe.Pointer
	if len(data) > 0 {
		dataPtr = unsafe.Pointer(&data[0])
	}

	var cError C.int
	var cMessage *C.char

	ptr := C.modterm_open_impl(
		dataPtr,
		C.size_t(len(data)),
		&cError,
		(**C.char)(unsafe.Pointer(&cMessage)),
	)

	if ptr == nil {
		if cMessage != nil {
			message := C.GoString(cMessage)
			C.openmpt_free_string(cMessage)

			return nil, fmt.Errorf("libopenmpt error %d: %s", int(cError), message)
		}

		return nil, fmt.Errorf("libopenmpt error %d", int(cError))
	}

	module := &Module{
		ptr: ptr,
		mu:  sync.Mutex{},
	}
	runtime.SetFinalizer(module, (*Module).Close)

	return module, nil
}

// Close destroys the module. It is safe to call more than once. The runtime
// finalizer calls it as well when the module becomes unreachable.
func (m *Module) Close() {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ptr == nil {
		return
	}

	C.modterm_close_impl(m.ptr)

	m.ptr = nil
	runtime.SetFinalizer(m, nil)
}

// Render produces at most sampleRate*seconds worth of interleaved stereo
// float32 audio. output must contain an even number of samples: two per
// frame (left, right). It returns the number of frames actually rendered,
// which is 0 once the song has ended.
func (m *Module) Render(sampleRate int, output []float32) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(output)%2 != 0 {
		return 0, fmt.Errorf("output buffer must contain an even number of samples")
	}

	frames := len(output) / 2
	if frames == 0 {
		return 0, nil
	}

	rendered := C.modterm_render_impl(
		m.ptr,
		C.int32_t(sampleRate),
		C.size_t(frames),
		(*C.float)(unsafe.Pointer(&output[0])),
	)

	return int(rendered), nil
}

// Metadata returns a single metadata value for key (e.g. "title", "artist",
// "tracker", "type_long", "message"). It returns "" when the key is unknown.
func (m *Module) Metadata(key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))

	value := C.modterm_metadata_impl(m.ptr, cKey)
	if value == nil {
		return "", nil
	}

	defer C.openmpt_free_string(value)
	return C.GoString(value), nil
}

// -- convenience metadata getters --

func (m *Module) Title() string {
	value, _ := m.Metadata("title")
	return value
}

func (m *Module) Artist() string {
	value, _ := m.Metadata("artist")
	return value
}

func (m *Module) Tracker() string {
	value, _ := m.Metadata("tracker")
	return value
}

func (m *Module) TypeLong() string {
	value, _ := m.Metadata("type_long")
	return value
}

func (m *Module) Message() string {
	value, _ := m.Metadata("message")
	return value
}

// Duration returns the total playing time. Some modules loop or never end; in
// that case libopenmpt reports an effectively infinite duration.
func (m *Module) Duration() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	s := C.modterm_duration_impl(m.ptr)

	return time.Duration(float64(s) * float64(time.Second))
}

// -- song structure --

func (m *Module) NumChannels() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return int(C.modterm_numchannels_impl(m.ptr))
}

func (m *Module) NumPatterns() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return int(C.modterm_numpatterns_impl(m.ptr))
}

// NumOrders returns the length of the module's sequence (not to be
// confused with the patterns themselves).
func (m *Module) NumOrders() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return int(C.modterm_numorders_impl(m.ptr))
}

// OrderPattern returns the pattern index played at the given order
// position (or -1 when the position is out of range).
func (m *Module) OrderPattern(order int) int {
	if order < 0 || order >= m.NumOrders() {
		return -1
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if C.modterm_orderplayable_impl(m.ptr, C.int32_t(order)) == 0 {
		return -1
	}

	return int(C.modterm_orderpattern_impl(m.ptr, C.int32_t(order)))
}

// IsOrderPlayable returns whether the given order position is an
// actual pattern (truthy) or a skip (+++)/stop(---) (falsy).
func (m *Module) IsOrderPlayable(order int) bool {
	return m.OrderPattern(order) >= 0
}

// Pattern returns a view over a single pattern of the module.
func (m *Module) Pattern(index int) (*Pattern, error) {
	if index < 0 || index >= m.NumPatterns() {
		return nil, fmt.Errorf("pattern index out of range: %d", index)
	}

	rows := m.patternRows(index)
	if rows < 0 {
		return nil, fmt.Errorf("could not get pattern %d", index)
	}

	return &Pattern{module: m, index: index, rows: rows}, nil
}

// patternRows exposes the pattern-row query to the (C-free) pattern.go file.
func (m *Module) patternRows(index int) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return int(C.modterm_patternrows_impl(m.ptr, C.int32_t(index)))
}

// patternCommand exposes the pattern cell query to the (C-free) pattern.go file.
func (m *Module) patternCommand(pattern, row, channel, command int) uint8 {
	m.mu.Lock()
	defer m.mu.Unlock()

	return uint8(C.modterm_patterncommand_impl(
		m.ptr,
		C.int32_t(pattern),
		C.int32_t(row),
		C.int32_t(channel),
		C.int(command),
	))
}

func (m *Module) ReadPattern(index int) (PatternData, error) {
	if m == nil {
		return PatternData{}, fmt.Errorf("nil module")
	}

	if index < 0 {
		return PatternData{}, fmt.Errorf("pattern index out of range: %d", index)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ptr == nil {
		return PatternData{}, fmt.Errorf("module is closed")
	}

	patternCount := int(C.modterm_numpatterns_impl(m.ptr))
	if index >= patternCount {
		return PatternData{}, fmt.Errorf("pattern index out of range: %d", index)
	}

	rows := int(C.modterm_patternrows_impl(m.ptr, C.int32_t(index)))
	channels := int(C.modterm_numchannels_impl(m.ptr))

	if rows < 0 || channels < 0 {
		return PatternData{}, fmt.Errorf("could not read pattern %d", index)
	}

	data := PatternData{
		Index: index,
		Rows:  rows,
		Cells: make([][]Cell, rows),
	}

	for row := 0; row < rows; row++ {
		data.Cells[row] = make([]Cell, channels)

		for channel := 0; channel < channels; channel++ {
			data.Cells[row][channel] = Cell{
				Note: uint8(C.modterm_patterncommand_impl(
					m.ptr,
					C.int32_t(index),
					C.int32_t(row),
					C.int32_t(channel),
					C.int(CommandNote),
				)),
				Instrument: uint8(C.modterm_patterncommand_impl(
					m.ptr,
					C.int32_t(index),
					C.int32_t(row),
					C.int32_t(channel),
					C.int(CommandInstrument),
				)),
				Effect: uint8(C.modterm_patterncommand_impl(
					m.ptr,
					C.int32_t(index),
					C.int32_t(row),
					C.int32_t(channel),
					C.int(CommandEffect),
				)),
				Volume: uint8(C.modterm_patterncommand_impl(
					m.ptr,
					C.int32_t(index),
					C.int32_t(row),
					C.int32_t(channel),
					C.int(CommandVolume),
				)),
				Parameter: uint8(C.modterm_patterncommand_impl(
					m.ptr,
					C.int32_t(index),
					C.int32_t(row),
					C.int32_t(channel),
					C.int(CommandParameter),
				)),
			}
		}
	}

	return data, nil
}

// -- playback position --

func (m *Module) CurrentOrder() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return int(C.modterm_currentorder_impl(m.ptr))
}

func (m *Module) CurrentPattern() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return int(C.modterm_currentpattern_impl(m.ptr))
}

func (m *Module) CurrentRow() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return int(C.modterm_currentrow_impl(m.ptr))
}

// SeekOrderRow jumps to the given order/row. It fails when the module cannot
// seek to that position.
func (m *Module) SeekOrderRow(order, row int) error {
	orders := m.NumOrders()
	if order < 0 || order >= orders {
		return fmt.Errorf("order %d is outside the module's %d orders", order, orders)
	}

	pattern := m.OrderPattern(order)
	if pattern < 0 {
		return fmt.Errorf("order %d holds no pattern", order)
	}

	rows := m.patternRows(pattern)
	switch {
	case row < 0:
		return fmt.Errorf("row %d is negative", row)

	case rows > 0 && row >= rows:
		return fmt.Errorf("row %d is past the end of pattern %d, which has %d rows",
			row, pattern, rows)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if ok := C.modterm_setpositionorderrow_impl(
		m.ptr,
		C.int32_t(order),
		C.int32_t(row),
	); ok == 0 {
		return fmt.Errorf("failed to seek order %d row %d", order, row)
	}

	return nil
}

type Position struct {
	Order   int
	Pattern int
	Row     int
}

func (m *Module) Position() Position {
	if m == nil {
		return Position{}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ptr == nil {
		return Position{}
	}

	return Position{
		Order:   int(C.modterm_currentorder_impl(m.ptr)),
		Pattern: int(C.modterm_currentpattern_impl(m.ptr)),
		Row:     int(C.modterm_currentrow_impl(m.ptr)),
	}
}
