package throttle

import (
	"context"
	"time"

	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/sync"
	"golang.org/x/time/rate"
)

var log = logging.LoggerForModule()

// DropThrottle assumes run is called with the same input function, such that calls above throttled rate can be thrown
// away. If calls are drops, another call of the input function will always happen afterwards.
type DropThrottle interface {
	// Run submits a function for execution, if possible. If it is dropped, `false` will be returned.
	Run(f func()) bool
}

// NewDropThrottle returns a new instance of a DropThrottle.
func NewDropThrottle(window time.Duration) DropThrottle {
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

// Run will run the input function now, after some period of time, or drop it (and return `false` in that case).
func (t *throttleImpl) Run(f func()) bool {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.limiter.Allow() {
		go f()
		return true
	}
	if !t.scheduled {
		t.scheduled = true
		go t.schedule(f)
		return true
	}
	return false
}

func (t *throttleImpl) schedule(f func()) {
	err := t.limiter.Wait(context.Background())
	if err != nil {
		log.Errorf("error waiting for throttle token: %s", err)
		return
	}

	t.lock.Lock()
	t.scheduled = false
	t.lock.Unlock()
	f()
}
