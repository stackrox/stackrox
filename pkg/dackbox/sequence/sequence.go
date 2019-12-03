package sequence

import "sync/atomic"

// Sequence is n object that provides a thread safe monotonically increasing integer value.
type Sequence func() uint64

// NewSequence returns a new instance of a Sequence that uses a simple atomic integer.
func NewSequence() Sequence {
	var value uint64
	return func() uint64 {
		return atomic.AddUint64(&value, 1)
	}
}
