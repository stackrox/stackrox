package ratetracker

import (
	"math"
	"time"
)

const (
	minimumRatePeriod = 10 * time.Second
)

// Inspiration is taken from estimated average recent request rate limiter (EARRRL):
// https://blog.jnbrymn.com/2021/03/18/estimated-average-recent-request-rate-limiter.html
//
// A shorter period will cause older rates to diminish more rapidly, while
// a longer period will result in older rates retaining greater importance.
// The tick is set to 1s. The minimum period is 10s.
type clusterRate struct {
	// index is required for Heap.
	index     int
	clusterID string

	lastUpdate  int64
	halfPeriod  float64
	ratePerSec  float64
	accumulator float64
}

func (cr *clusterRate) receiveMsg() {
	now := time.Now().Unix()

	cr.accumulator *= math.Exp(float64(cr.lastUpdate-now) / cr.halfPeriod)
	cr.accumulator++
	cr.ratePerSec = cr.accumulator / cr.halfPeriod

	cr.lastUpdate = now
}

func newClusterRate(clusterID string, period time.Duration) *clusterRate {
	if period < minimumRatePeriod {
		period = minimumRatePeriod
	}

	return &clusterRate{
		index:      -1,
		clusterID:  clusterID,
		halfPeriod: period.Seconds() / 2.0,
		lastUpdate: time.Now().Unix(),
		ratePerSec: 0.0,
	}
}
