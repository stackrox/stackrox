package ratetracker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClusterRateFormula(t *testing.T) {
	tests := []struct {
		name   string
		period time.Duration

		// We can use the negative time difference (timeDiff) to simulate
		// the rate at which messages are received.
		timeDiff   time.Duration
		iterations int
		rate       float64
		rateDelta  float64
	}{
		{"10 messages in burst", time.Minute, time.Nanosecond, 10, 0.3333, 0.0001},
		{"10 messages in burst", time.Second, time.Nanosecond, 10, 2, 0.0001},
		{"1 message per min", time.Minute, -time.Minute, 10, 0.0385, 0.0001},
		{"1 message per 2 mins", time.Minute, -2 * time.Minute, 10, 0.0339, 0.0001},
		{"1 message per min for period of 10 mins", 10 * time.Minute, -time.Minute, 10, 0.0159, 0.0001},
		{"1 message per 2 mins for period of 10 mins", 10 * time.Minute, -2 * time.Minute, 10, 0.0099, 0.0001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := newClusterRate("c1", tt.period)
			assert.Equal(t, 0.0, cr.ratePerSec)

			for i := 0; i < 10; i++ {
				cr.lastUpdate = time.Now().Add(tt.timeDiff).Unix()
				cr.receiveMsg()
			}
			assert.InDelta(t, tt.rate, cr.ratePerSec, tt.rateDelta)
		})
	}
}

func TestNewClusterRate(t *testing.T) {
	now := time.Now().Unix()
	cr := newClusterRate("c1", time.Minute)

	assert.Equal(t, "c1", cr.clusterID)
	assert.Equal(t, -1, cr.index)
	assert.Equal(t, 0.0, cr.ratePerSec)
	assert.Equal(t, time.Minute.Seconds()/2.0, cr.halfPeriod)
	assert.InDelta(t, now, cr.lastUpdate, 5)

	crMin := newClusterRate("c1", time.Nanosecond)
	assert.Equal(t, minimumRatePeriod.Seconds()/2.0, crMin.halfPeriod)
}

/*** Benchmark tests ***/

func BenchmarkClusterRateReceiveMsg(b *testing.B) {
	tests := []struct {
		name     string
		lastTime int64
	}{
		{"time ticks", time.Now().Add(-10 * time.Hour).Unix()},
		{"no time ticks", time.Now().Add(10 * time.Hour).Unix()},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			n := newClusterRate("c1", 10*time.Hour)
			for i := 0; i < b.N; i++ {
				n.lastUpdate = tt.lastTime
				n.receiveMsg()
			}
		})
	}
}
