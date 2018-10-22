package processsignal

import (
	"fmt"
	"time"

	"github.com/hashicorp/golang-lru"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/sensor/common/metrics"
	"golang.org/x/time/rate"
)

const (
	cacheSize = 100000

	burstMax      = 10
	limitDuration = 30 * time.Minute
)

type deduper struct {
	cache *lru.Cache
}

func newDeduper() *deduper {
	c, err := lru.New(cacheSize)
	if err != nil {
		panic(err)
	}
	return &deduper{
		cache: c,
	}
}

func generateProcessSignalKey(signal *v1.ProcessSignal) string {
	return fmt.Sprintf("%s %s %s %s", signal.GetContainerId(), signal.GetExecFilePath(), signal.GetName(), signal.GetArgs())
}

func (d *deduper) Allow(signal *v1.ProcessSignal) (allow bool) {
	defer func() {
		if allow {
			metrics.IncrementProcessDedupeCacheMisses()
		} else {
			metrics.IncrementProcessDedupeCacheHits()
		}
	}()

	key := generateProcessSignalKey(signal)
	elem, ok := d.cache.Get(key)
	if !ok {
		allow = true
		// Add nil the first time, this is a bit of savings if the unique process name only shows up once
		// because we won't allocate a rate limiter
		d.cache.Add(key, nil)
		return
	}
	// If elem is nil and ok is true, then this is the second time at this point so allocate a rate limiter and use it
	var limiter *rate.Limiter
	if elem == nil {
		limiter = rate.NewLimiter(rate.Every(limitDuration), burstMax)
		d.cache.Add(key, limiter)
	} else {
		limiter = elem.(*rate.Limiter)
	}

	allow = limiter.Allow()
	return
}
