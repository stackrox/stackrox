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

// NewSemaphoreWithValue creates a semaphore wrapper that implements the scanner interface with the given value
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

// NodeScanSemaphore is an interface that implements part of the node scanner interface
type NodeScanSemaphore interface {
	MaxConcurrentNodeScanSemaphore() *semaphore.Weighted
}

// NewNodeSemaphoreWithValue creates a semaphore wrapper that implements the node scanner interface with the given value
func NewNodeSemaphoreWithValue(val int64) NodeScanSemaphore {
	return &nodeScanSemaphoreImpl{
		ScanSemaphore: NewSemaphoreWithValue(val),
	}
}

type nodeScanSemaphoreImpl struct {
	ScanSemaphore
}

func (n *nodeScanSemaphoreImpl) MaxConcurrentNodeScanSemaphore() *semaphore.Weighted {
	return n.MaxConcurrentScanSemaphore()
}

// NodeMatchSemaphore sets the maximum number of concurrent node match operations performed
type NodeMatchSemaphore interface {
	MaxConcurrentNodeMatchSemaphore() *semaphore.Weighted
}

func NewNodeMatchSemaphoreWithValue(val int64) NodeMatchSemaphore {
	return &nodeMatchSemaphoreImpl{
		NewSemaphoreWithValue(val),
	}
}

type nodeMatchSemaphoreImpl struct {
	ScanSemaphore
}

func (n *nodeMatchSemaphoreImpl) MaxConcurrentNodeMatchSemaphore() *semaphore.Weighted {
	return n.MaxConcurrentScanSemaphore()
}
