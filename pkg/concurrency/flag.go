package concurrency

import "sync/atomic"

// Flag is an atomic boolean flag.
type Flag struct {
	val uint32
}

// Get gets the current value of the flag.
func (f *Flag) Get() bool {
	return atomic.LoadUint32(&f.val)&0x1 != 0
}

// Set sets the value of the flag, discarding the previous value.
func (f *Flag) Set(v bool) {
	atomic.StoreUint32(&f.val, b2i(v))
}

// TestAndSet sets the value of the flag, and returns the *previous* value of the flag.
func (f *Flag) TestAndSet(v bool) bool {
	return atomic.SwapUint32(&f.val, b2i(v))&0x1 != 0
}

// Toggle flips the value of the flag, and returns the *new* value of the flag.
func (f *Flag) Toggle() bool {
	return atomic.AddUint32(&f.val, 1)&0x1 != 0
}

func b2i(v bool) uint32 {
	if v {
		return 1
	}
	return 0
}
