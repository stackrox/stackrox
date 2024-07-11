package concurrency

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
)

var (
	err1 = errors.New("testerr1")
	err2 = errors.New("testerr2")
)

func TestErrorSignal_ZeroValueIsTriggered(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	var s ErrorSignal
	a.True(s.IsDone(), "error signal should be triggered")

	err, ok := s.Error()
	a.NoError(err, "zero value should return a nil error")
	a.True(ok, "zero value should indicate it is triggered")

	a.True(s.Reset(), "reset on zero value signal should work")
	a.False(s.IsDone(), "signal should not be done after reset")
}

func TestNewErrorSignalIsNotDone(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	s := NewErrorSignal()
	a.False(s.IsDone(), "error signal should not be triggered")

	err, ok := s.Error()
	a.Nil(err, "Error() should return a nil error")
	a.False(ok, "Error() should return a false boolean value")
}

func TestNewErrorSignalResetHasNoEffect(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	s := NewErrorSignal()
	wc := s.WaitC()
	a.False(s.Reset(), "Reset on a new error signal should return false")
	a.False(s.IsDone(), "error signal should not be triggered")
	a.Equal(wc, s.WaitC(), "the channel should not change when reset has no effect")

	err, ok := s.Error()
	a.Nil(err, "Error() should return a nil error")
	a.False(ok, "Error() should return a false boolean value")
}

func TestErrorSignalTrigger(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	s := NewErrorSignal()
	a.False(s.IsDone(), "signal should not be triggered")
	wc := s.WaitC()

	a.True(s.SignalWithError(err1), "calling SignalWithError should return true")
	a.True(s.IsDone(), "error signal should be triggered")
	a.True(IsDone(wc), "the old wait channel should be closed")

	err, ok := s.Error()
	a.Equal(err1, err, "Error() should return err1")
	a.True(ok, "Error() should return a true boolean value")

	// Test that Signal() can be called repeatedly
	a.False(s.SignalWithError(err2), "calling SignalWithError the second time should return false")
	a.True(s.IsDone(), "error signal should be triggered")

	// Test that err1 is still returned, not err2
	err, ok = s.Error()
	a.Equal(err1, err, "Error() should return err1")
	a.True(ok, "Error() should return a true boolean value")
}

func TestErrorSignalTriggerTwiceWithReset(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	s := NewErrorSignal()
	wc := s.WaitC()
	a.True(s.SignalWithError(err1), "calling SignalWithError should return true")
	a.True(s.IsDone(), "error signal should be triggered")
	a.True(IsDone(wc), "old wait channel should be closed")

	err, ok := s.Error()
	a.Equal(err1, err, "Error() should return err1")
	a.True(ok, "Error() should return a true boolean value")

	a.True(s.Reset(), "calling Reset on a triggered error signal should return true")
	a.False(s.IsDone(), "signal should not be triggered after reset")
	a.True(IsDone(wc), "old wait channel should still be closed")

	err, ok = s.Error()
	a.Nil(err, "Error() should return a nil error after reset")
	a.False(ok, "Error() should return a false boolean valueafter reset")

	a.False(s.Reset(), "calling reset a second time should return false")

	wc = s.WaitC()
	a.True(s.SignalWithError(err2), "calling SignalWithError should return true")
	a.True(s.IsDone(), "error signal should be triggered")
	a.True(IsDone(wc), "old wait channel should be closed")

	err, ok = s.Error()
	a.Equal(err2, err, "Error() should return err1")
	a.True(ok, "Error() should return a true boolean value")
}

// Tests that every error that is passed to a *successful* invocation of SignalWithError() is observed by exactly one
// invocation of ErrorAndReset.
func TestErrorSignal_SignalAndResetAreAtomic(t *testing.T) {
	t.Parallel()

	errSig := NewErrorSignal()

	var triggeredErrs, resetErrs []error
	var mutex sync.Mutex

	var initWG, doneWG sync.WaitGroup
	// this is used to ensure all goroutines start at about the same time, independent of the order in which they
	// were spawned.
	initWG.Add(1)

	numGoroutines := 100000
	if buildinfo.RaceEnabled {
		// Race detector has a limit of 8128 goroutines.
		numGoroutines = 1000
	}

	for i := 0; i < numGoroutines; i++ {
		doneWG.Add(1)
		iCopy := i
		go func() {
			defer doneWG.Done()
			initWG.Wait()

			var errSlice *[]error
			var err error
			if iCopy%2 != 0 {
				var ok bool
				err, ok = errSig.ErrorAndReset()
				if !ok {
					return
				}
				errSlice = &resetErrs

			} else {
				err = fmt.Errorf("error %d", iCopy)
				if !errSig.SignalWithError(err) {
					return
				}
				errSlice = &triggeredErrs
			}

			mutex.Lock()
			defer mutex.Unlock()
			*errSlice = append(*errSlice, err)
		}()
	}

	initWG.Done()

	doneWG.Wait()
	if err, ok := errSig.ErrorAndReset(); ok {
		resetErrs = append(resetErrs, err)
	}

	assert.ElementsMatch(t, triggeredErrs, resetErrs)
}

func TestErrorSignalWhenIsTriggered(t *testing.T) {
	firstSig := NewErrorSignal()
	secondSig := NewErrorSignal()

	var retVal bool
	var signalWhenReturned Flag
	go func() {
		retVal = firstSig.SignalWhen(&secondSig, Never())
		signalWhenReturned.Set(true)
	}()

	fakeErr := errors.New("FakeErr")

	assert.False(t, firstSig.IsDone())

	secondSig.SignalWithError(fakeErr)

	assert.True(t, WaitWithTimeout(&firstSig, time.Second))
	poller := NewPoller(signalWhenReturned.Get, 10*time.Millisecond)
	assert.True(t, WaitWithTimeout(poller, time.Second))
	assert.Equal(t, firstSig.Err(), fakeErr)
	assert.Equal(t, true, retVal)
}

func TestErrorSignalWhenSignalItselfIsTriggered(t *testing.T) {
	firstSig := NewErrorSignal()
	secondSig := NewErrorSignal()

	var retVal bool
	var signalWhenReturned Flag
	go func() {
		retVal = firstSig.SignalWhen(&secondSig, Never())
		signalWhenReturned.Set(true)
	}()

	assert.False(t, firstSig.IsDone())
	firstSig.Signal()
	assert.True(t, firstSig.IsDone())

	poller := NewPoller(signalWhenReturned.Get, 10*time.Millisecond)
	assert.True(t, WaitWithTimeout(poller, time.Second))

	assert.False(t, secondSig.IsDone())
	assert.Equal(t, false, retVal)
}

func TestErrorSignalWhenWithCancel(t *testing.T) {
	firstSig := NewErrorSignal()
	secondSig := NewErrorSignal()

	ctx, cancel := context.WithCancel(context.Background())

	var retVal bool
	var signalWhenReturned Flag
	go func() {
		retVal = firstSig.SignalWhen(&secondSig, ctx)
		signalWhenReturned.Set(true)
	}()
	assert.False(t, firstSig.IsDone())
	cancel()
	assert.False(t, firstSig.IsDone())
	assert.False(t, secondSig.IsDone())

	poller := NewPoller(signalWhenReturned.Get, 10*time.Millisecond)
	assert.True(t, WaitWithTimeout(poller, time.Second))

	assert.Equal(t, false, retVal)
}
