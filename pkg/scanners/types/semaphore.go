package types

import "golang.org/x/sync/semaphore"

// ScanSemaphore is an interface that implements part of the scanner interface
type ScanSemaphore interface {
	MaxConcurrentScanSemaphore() *semaphore.Weighted
}

// NewDefaultSemaphore creates a semaphore wrapper that implements the scanner interface with a value of 6
func NewDefaultSemaphore() ScanSemaphore {
	return &scanSemaphoreImpl{
		sema: semaphore.NewWeighted(6),
	}
}

// NewSemaphoreWithValue creates a semaphore wrapper from the input max
// that implements the scanner interface with a value of 6
func NewSemaphoreWithValue(val int64) ScanSemaphore {
	return &scanSemaphoreImpl{
		sema: semaphore.NewWeighted(val),
	}
}

type scanSemaphoreImpl struct {
	sema *semaphore.Weighted
}

func (s *scanSemaphoreImpl) MaxConcurrentScanSemaphore() *semaphore.Weighted {
	return s.sema
}
