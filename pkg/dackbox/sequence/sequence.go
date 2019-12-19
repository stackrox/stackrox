package sequence

import "sync/atomic"

// Sequence is n object that provides a thread safe monotonically increasing integer value.
type Sequence interface {
	Load() uint64
	Add() uint64
}

// NewSequence returns a new instance of a Sequence that uses a simple atomic integer.
func NewSequence() Sequence {
	return &sequenceImpl{}
}

type sequenceImpl struct {
	value uint64
}

func (s *sequenceImpl) Load() uint64 {
	return atomic.LoadUint64(&s.value)
}

func (s *sequenceImpl) Add() uint64 {
	return atomic.AddUint64(&s.value, 1)
}
