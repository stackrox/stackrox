package reprocessor

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/time/rate"
)

// Assumes run is called with the same input function, such that calls above throttled rate can be thrown away.
type throttle interface {
	run(f func())
}

func newThrottle(window time.Duration) throttle {
	return &throttleImpl{
		limiter:   rate.NewLimiter(rate.Every(window), 1),
		scheduled: false,
	}
}

type throttleImpl struct {
	lock      sync.Mutex
	limiter   *rate.Limiter
	scheduled bool
}

func (t *throttleImpl) run(f func()) {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.limiter.Allow() {
		go f()
	} else if !t.scheduled {
		t.scheduled = true
		go t.schedule(f)
	}
}

func (t *throttleImpl) schedule(f func()) {
	err := t.limiter.Wait(context.Background())
	if err != nil {
		log.Errorf("error waiting for scheduled reprocessing of risk: %s", err)
		return
	}

	t.lock.Lock()
	t.scheduled = false
	t.lock.Unlock()
	f()
}
