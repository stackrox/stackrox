package rate

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
)

func TestRateManager(t *testing.T) {
	recordC := make(chan struct{})
	defer close(recordC)
	limitExceededSignal := make(chan struct{})
	defer close(limitExceededSignal)
	limitMissedSignal := make(chan struct{})
	defer close(limitMissedSignal)
	errSignal := concurrency.NewErrorSignal()
	manager := &managerImpl{
		recordsC:  recordC,
		rateLimit: 10,
		onLimitExceeded: func(_ int) {
			select {
			case limitExceededSignal <- struct{}{}:
			case <-time.After(500 * time.Millisecond):
				t.Error("timeout")
				t.FailNow()
			}
		},
		stopC: &errSignal,
		onLimitMissed: func(_ int) {
			select {
			case limitMissedSignal <- struct{}{}:
			case <-time.After(500 * time.Millisecond):
				t.Error("timeout")
				t.FailNow()
			}
		},
	}
	ticker := make(chan time.Time)
	defer close(ticker)
	go manager.run(ticker)

	for i := 0; i < 10; i++ {
		manager.Record()
	}
	ticker <- time.Now()
	select {
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout waiting for call")
		t.FailNow()
	case <-limitExceededSignal:
	}
	for i := 0; i < 10; i++ {
		manager.Record()
	}
	ticker <- time.Now()
	select {
	case <-time.After(500 * time.Millisecond):
	case <-limitExceededSignal:
		t.Error("should not trigger a second call")
		t.FailNow()
	}
	for i := 0; i < 5; i++ {
		manager.Record()
	}
	ticker <- time.Now()
	select {
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout waiting for call")
		t.FailNow()
	case <-limitMissedSignal:
	}
}
