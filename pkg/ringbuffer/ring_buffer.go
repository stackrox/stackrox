package ringbuffer

import (
	"github.com/pkg/errors"
)

const (
	defaultRingBufferCapacity = 8192
)

// A RingBuffer implements a cyclical buffer, that maintains the last cap bytes written to it, where cap is the capacity
// of the ring buffer.
// This type is not safe to be used concurrently. When using it from multiple goroutines, all accesses have to be
// synchronized externally.
type RingBuffer struct {
	buf      []byte
	startOfs int
	fill     int
}

// NewRingBuffer creates and returns a new ring buffer with the given capacity.
func NewRingBuffer(capacity int) *RingBuffer {
	if capacity < 0 {
		panic(errors.Errorf("invalid ring buffer capacity %d", capacity))
	} else if capacity == 0 {
		capacity = defaultRingBufferCapacity
	}
	return &RingBuffer{
		buf: make([]byte, capacity),
	}
}

// Capacity returns the capacity of the ring buffer.
func (r *RingBuffer) Capacity() int {
	return len(r.buf)
}

// Size returns the current size of the ring buffer.
func (r *RingBuffer) Size() int {
	return r.fill
}

// Reset clears the ring buffer. The callback, if given, is invoked for all data chunks that are cleared from the
// buffer.
func (r *RingBuffer) Reset(cb func([]byte)) {
	if r.fill > 0 && cb != nil {
		for _, chunk := range r.ReadAll() {
			cb(chunk)
		}
	}

	r.startOfs = 0
	r.fill = 0
}

// ReadAll returns all data stored in the ring buffer, possibly in chunks. The returned chunks are only valid until the
// next call to either `Write` or `Reset`.
func (r *RingBuffer) ReadAll() [][]byte {
	return r.readRaw(r.startOfs, r.fill)
}

// readRaw reads and returns up to `num` bytes of data starting at `startIdx`. As this function is not exposed and only
// called externally, no further validation on the input arguments is performed.
func (r *RingBuffer) readRaw(startIdx, num int) [][]byte {
	if num <= 0 {
		return nil
	}

	endIdx := startIdx + num
	if endIdx > len(r.buf) {
		return [][]byte{r.buf[startIdx:], r.buf[:endIdx-len(r.buf)]}
	}
	return [][]byte{r.buf[startIdx:endIdx]}
}

// ReadFirst returns the first num bytes stored in the ring buffer, possibly in chunks. The returned chunks are only
// valid until the next call to either `Write` or `Reset`.
func (r *RingBuffer) ReadFirst(num int) [][]byte {
	if num > r.fill {
		num = r.fill
	}

	return r.readRaw(r.startOfs, num)
}

// Read reads up to num bytes from the ring buffer, starting at a position determined by `from`. If `from` is negative,
// it is interpreted to refer to the position `-from` bytes from the end of the buffer (clamped at position 0, i.e., the
// beginning of the buffer). The returned chunks are only valid until the next call to either `Write` or `Reset`.
func (r *RingBuffer) Read(from, num int) [][]byte {
	if from < 0 {
		from = r.fill + from
		if from < 0 {
			from = 0
		}
	}
	if from >= r.fill {
		return nil
	}
	if num > r.fill-from {
		num = r.fill - from
	}

	return r.readRaw((r.startOfs+from)%len(r.buf), num)
}

// ReadLast returns the last num bytes in the ring buffer, possibly in chunks. The returned chunks are only valid until
// the next call to either `Write` or `Reset`.
func (r *RingBuffer) ReadLast(num int) [][]byte {
	if num > r.fill {
		num = r.fill
	}

	return r.readRaw((r.startOfs+r.fill-num)%len(r.buf), num)
}

// Write writes data to the ring buffer, evicting old data if the buffer is full. The callback cb is called for every
// chunk of data that is evicted from the ring buffer, or skipped in the input data because it would not fit. The caller
// must not retain a reference to the data after the callback completes; if the data needs to be retained, it must be
// copied by the caller.
func (r *RingBuffer) Write(data []byte, cb func([]byte)) {
	if len(data) >= len(r.buf) {
		if cb != nil {
			for _, chunk := range r.ReadAll() {
				cb(chunk)
			}
		}

		overflowLen := len(data) - len(r.buf)
		if overflowLen > 0 && cb != nil {
			cb(data[:overflowLen])
		}

		copy(r.buf, data[overflowLen:])
		r.startOfs = 0
		r.fill = len(r.buf)
		return
	}

	if overflowLen := len(data) + r.fill - len(r.buf); overflowLen > 0 {
		if cb != nil {
			for _, chunk := range r.ReadFirst(overflowLen) {
				cb(chunk)
			}
		}

		r.startOfs = (r.startOfs + overflowLen) % len(r.buf)
		r.fill -= overflowLen
	}

	// We can now assume that the buffer has enough capacity for data
	startIdx := (r.startOfs + r.fill) % len(r.buf)

	dataLen := len(data)

	endIdx := startIdx + len(data)
	if over := endIdx - len(r.buf); over > 0 {
		copy(r.buf[startIdx:], data[:len(data)-over])
		startIdx = 0
		endIdx = over
		data = data[len(data)-over:]
	}
	copy(r.buf[startIdx:endIdx], data)

	r.fill += dataLen
}
