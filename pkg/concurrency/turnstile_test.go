package concurrency

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test that a waiting go routine is released with true returned when AllowOne is called.
func TestWaitSucceeds(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	turnstile := NewTurnstile()

	waitCalled := NewSignal()
	callChecked := NewSignal()
	c := &callable{}
	go func() {
		if waitCalled.Signal() && turnstile.Wait() {
			c.call()
		}
		callChecked.Signal()
	}()
	waitCalled.Wait()

	turnstile.AllowOne()

	callChecked.Wait()
	a.True(c.wasCalled, "callable should be called")
	a.True(turnstile.Close(), "close should close the turnstile")
}

// Test that a waiting go routine is released with false returned when Close is called.
func TestWaitWhenClosed(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	turnstile := NewTurnstile()

	callChecked := NewSignal()
	c := &callable{}
	go func() {
		if !turnstile.Wait() {
			c.call()
		}
		callChecked.Signal()
	}()

	a.True(turnstile.Close(), "close should close the turnstile")
	callChecked.Wait()
	a.True(c.wasCalled, "callable should be called")
}

// Test that a waiting go routine is released with false when the cancelWhen Waitable object expires.
func TestWaitWithCancel(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	turnstile := NewTurnstile()

	// Create a preloaded wait channel so that the wait expires immediately.
	cancelChan := make(chan struct{}, 1)
	cancelChan <- struct{}{}
	cancelWhen := WaitableChan(cancelChan)

	callChecked := NewSignal()
	c := &callable{}
	go func() {
		if !turnstile.WaitWithCancel(cancelWhen) {
			c.call()
		}
		callChecked.Signal()
	}()

	callChecked.Wait()
	a.True(c.wasCalled, "callable should be called")
	a.True(turnstile.Close(), "close should close the turnstile")
}

// Test that closing more than once returns false.
func TestRepeateCloseReturnsFalse(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	turnstile := NewTurnstile()

	a.True(turnstile.Close(), "close should close the turnstile")
	a.False(turnstile.Close(), "close should do nothing and return false")
	a.False(turnstile.Close(), "close should do nothing and return false")
}

// Test helper.
///////////////

type callable struct {
	wasCalled bool
}

func (c *callable) call() {
	c.wasCalled = true
}
