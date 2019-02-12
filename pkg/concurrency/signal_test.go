package concurrency

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewSignalIsNotDone(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	s := NewSignal()
	a.False(s.IsDone(), "signal should not be triggered")
}

func TestNewSignalResetHasNoEffect(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	s := NewSignal()
	wc := s.WaitC()
	a.False(s.Reset(), "Reset on a new signal should return false")
	a.False(s.IsDone(), "signal should not be triggered")
	a.Equal(wc, s.WaitC(), "the channel should not change when reset has no effect")
}

func TestSignalTrigger(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	s := NewSignal()
	a.False(s.IsDone(), "signal should not be triggered")
	wc := s.WaitC()

	a.True(s.Signal(), "calling signal should return true")
	a.True(s.IsDone(), "signal should be triggered")
	a.True(IsDone(wc), "the old wait channel should be closed")

	// Test that Signal() can be called repeatedly
	a.False(s.Signal(), "calling signal the second time should return false")
	a.True(s.IsDone(), "signal should be triggered")
}

func TestSignalTriggerAndReset(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	s := NewSignal()
	wc := s.WaitC()
	a.True(s.Signal(), "calling signal should return true")
	a.True(s.IsDone(), "signal should be triggered")
	a.True(IsDone(wc), "old wait channel should be closed")

	a.True(s.Reset(), "calling Reset on a triggered signal should return true")
	a.False(s.IsDone(), "signal should not be triggered after reset")
	a.True(IsDone(wc), "old wait channel should still be closed")

	a.False(s.Reset(), "calling reset a second time should return false")
}

func TestSignalDoWithTimeout(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	var done bool
	action := func() {
		done = true
	}

	s := NewSignal()
	a.False(DoWithTimeout(&s, action, 100*time.Millisecond))
	a.False(done)
	go func() {
		time.Sleep(50 * time.Millisecond)
		s.Signal()
	}()
	a.True(DoWithTimeout(&s, action, 100*time.Millisecond))
	a.True(done)
}
