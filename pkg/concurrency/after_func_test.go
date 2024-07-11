package concurrency

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAfterFuncNoCancel(t *testing.T) {
	var done Flag
	AfterFunc(100*time.Millisecond, func() {
		done.Set(true)
	}, Never())
	assert.True(t, PollWithTimeout(done.Get, 10*time.Millisecond, 200*time.Millisecond))
}

func TestAfterFuncCancel(t *testing.T) {
	var done Flag
	sig := NewSignal()
	AfterFunc(50*time.Millisecond, func() {
		done.Set(true)
	}, &sig)
	sig.Signal()

	time.Sleep(100 * time.Millisecond)
	assert.False(t, done.Get())
}
