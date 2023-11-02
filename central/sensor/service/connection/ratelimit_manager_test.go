package connection

import (
	"container/heap"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRateLimitManagerDefaultMaxInitSync(t *testing.T) {
	m := newRateLimitManager()
	assert.Equal(t, math.MaxInt, m.maxSensors)

	assert.True(t, m.AddInitSync("test-1"), "Can add if limit is set to 0")
	assert.Len(t, m.initSyncSensors, 1)

	m.RemoveInitSync("test-1")
	assert.Len(t, m.initSyncSensors, 0)
}

func TestNewRateLimitManagerNegativeMaxInitSync(t *testing.T) {
	t.Setenv(env.CentralMaxInitSyncSensors.EnvVar(), "-1")
	assert.Panics(t, func() { newRateLimitManager() })
}

func TestNewRateLimitManagerZeroMaxInitSync(t *testing.T) {
	t.Setenv(env.CentralMaxInitSyncSensors.EnvVar(), "0")
	m := newRateLimitManager()
	assert.Equal(t, math.MaxInt, m.maxSensors)

	assert.True(t, m.AddInitSync("test-1"), "Can add if limit is set to 0")
	assert.Len(t, m.initSyncSensors, 1)

	m.RemoveInitSync("test-1")
	assert.Len(t, m.initSyncSensors, 0)
}

func TestNewRateLimitManagerMaxInitSync(t *testing.T) {
	t.Setenv(env.CentralMaxInitSyncSensors.EnvVar(), "3")
	m := newRateLimitManager()

	for i := 0; i < 3; i++ {
		assert.True(t, m.AddInitSync(fmt.Sprintf("test-%d", i)))
	}
	assert.False(t, m.AddInitSync("test-a"), "Unable to add after limit is reached")
	assert.Len(t, m.initSyncSensors, 3)

	m.RemoveInitSync("test-a")
	assert.False(t, m.AddInitSync("test-a"), "Unable to add after removing non-existing")

	m.RemoveInitSync("test-1")
	assert.Len(t, m.initSyncSensors, 2)
	assert.True(t, m.AddInitSync("test-a"), "Can add after one is removed")
	assert.Len(t, m.initSyncSensors, 3)

	assert.False(t, m.AddInitSync("test-b"), "Unable to add after limit is reached")
}

func TestInitSyncNilGuards(t *testing.T) {
	var m *rateLimitManager

	assert.Nil(t, m)
	assert.True(t, m.AddInitSync("test-1"))
	assert.NotPanics(t, func() { m.RemoveInitSync("test-1") })

	m = &rateLimitManager{
		maxSensors: 1,
	}
	assert.Nil(t, m.msgRateLimiter)
	assert.True(t, m.AddInitSync("test-1"))
	assert.False(t, m.AddInitSync("test-2"))

	assert.NotPanics(t, func() { m.RemoveInitSync("test-1") })
	assert.True(t, m.AddInitSync("test-2"))
}

func TestNewRateLimitManagerDefaultEventsPerSecond(t *testing.T) {
	m := newRateLimitManager()

	for i := 0; i < 100; i++ {
		require.False(t, m.LimitClusterMsg("c1"), "No limit")
	}
}

func TestNewRateLimitManagerNegativeEventsPerSecond(t *testing.T) {
	t.Setenv(env.CentralSensorMaxEventsPerSecond.EnvVar(), "-1")
	assert.Panics(t, func() { newRateLimitManager() })
}

func TestNewRateLimitManagerZeroEventsPerSecond(t *testing.T) {
	t.Setenv(env.CentralSensorMaxEventsPerSecond.EnvVar(), "0")
	m := newRateLimitManager()

	for i := 0; i < 100; i++ {
		require.False(t, m.LimitClusterMsg("c1"), "No limit")
	}
}

func TestNewRateLimitManagerEventsPerSecond(t *testing.T) {
	t.Setenv(env.CentralSensorMaxEventsPerSecond.EnvVar(), "3")
	t.Setenv(env.CentralSensorMaxEventsThrottleDuration.EnvVar(), "0")
	m := newRateLimitManager()

	hitLimit := false
	for i := 0; i < 30; i++ {
		limitMsg := m.LimitClusterMsg("c1")
		if i < 3 {
			require.False(t, limitMsg, "Limit is not reached")
			continue
		}

		if limitMsg {
			hitLimit = true
			break
		}
	}
	assert.True(t, hitLimit)

	// Wait for rate limit to refill.
	time.Sleep(time.Second)

	assert.False(t, m.LimitClusterMsg("c1"), "Rate is below threshold")
}

func TestLimitMsgNilGuards(t *testing.T) {
	var m *rateLimitManager

	assert.Nil(t, m)
	assert.False(t, m.LimitClusterMsg("c1"))

	m = &rateLimitManager{}
	assert.Nil(t, m.msgRateLimiter)
	assert.False(t, m.LimitClusterMsg("c1"))
}

// Test clusterMsgRate
func TestClusterMsgEwmaRateNoTick(t *testing.T) {
	tests := []struct {
		name       string
		period     time.Duration
		timeDiff   time.Duration
		iterations int
		rate       float64
		rateDelta  float64
	}{
		{"10 messages in burst", time.Minute, time.Hour, 10, 0.3333, 0.0001},
		{"10 messages in burst", time.Second, time.Hour, 10, 2, 0.0001},
		{"1 message per min", time.Minute, -time.Minute, 10, 0.0385, 0.0001},
		{"1 message per 2 mins", time.Minute, -2 * time.Minute, 10, 0.0339, 0.0001},
		{"1 message per min for period of 10 mins", 10 * time.Minute, -time.Minute, 10, 0.0159, 0.0001},
		{"1 message per 2 mins for period of 10 mins", 10 * time.Minute, -2 * time.Minute, 10, 0.0099, 0.0001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgRate := newClusterMsgRate("c1", tt.period)
			assert.Equal(t, 0.0, msgRate.rate)

			for i := 0; i < 10; i++ {
				msgRate.lastTime = time.Now().Add(tt.timeDiff).Unix()
				msgRate.recvMsg()
			}
			assert.InDelta(t, tt.rate, msgRate.rate, tt.rateDelta)
		})
	}
}

func TestHeapWorks(t *testing.T) {
	clusters := []struct {
		name    string
		numMsgs int
	}{
		{"c1", 1},
		{"c2", 8},
		{"c3", 9},
		{"c4", 5},
		{"c5", 3},
	}

	rl := newRateLimitManager()
	for i := 0; i < 10; i++ {
		for c := 0; c < len(clusters); c++ {
			if clusters[c].numMsgs > 0 {
				rl.LimitClusterMsg(clusters[c].name)
				clusters[c].numMsgs--
			}
		}
	}

	var throttleCandidates []string
	for len(*rl.clusterMsgRatesHeap) > 0 {
		throttleCandidates = append(throttleCandidates, heap.Pop(rl.clusterMsgRatesHeap).(*clusterMsgRate).clusterID)
	}

	assert.ElementsMatch(t, []string{"c4", "c3", "c2"}, throttleCandidates)
}

/*** Benchmark tests ***/

func BenchmarkClusterMsgRateRecvMsg(b *testing.B) {
	tests := []struct {
		name     string
		lastTime int64
	}{
		{"time ticks", time.Now().Add(-10 * time.Hour).Unix()},
		{"no time ticks", time.Now().Add(10 * time.Hour).Unix()},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			n := newClusterMsgRate("c1", 10*time.Hour)
			for i := 0; i < b.N; i++ {
				n.lastTime = tt.lastTime
				n.recvMsg()
			}
		})
	}
}

func getListOfClusters(n int) []string {
	var clusters []string
	for i := 0; i < n; i++ {
		clusters = append(clusters, fmt.Sprintf("cluster-%d", i))
	}

	return clusters
}

func BenchmarkLimitClusterMsg(b *testing.B) {
	tests := []struct {
		name         string
		lastRecv     time.Time
		clusters     int
		rl           *rateLimitManager
		clustersList []string
	}{
		{"00001 clusters", time.Now().Add(-time.Hour), 1, newRateLimitManager(), getListOfClusters(1)},
		{"00010 clusters", time.Now().Add(-time.Hour), 10, newRateLimitManager(), getListOfClusters(10)},
		{"00100 clusters", time.Now().Add(-time.Hour), 100, newRateLimitManager(), getListOfClusters(100)},
		{"01000 clusters", time.Now().Add(-time.Hour), 1000, newRateLimitManager(), getListOfClusters(1000)},
		{"10000 clusters", time.Now().Add(-time.Hour), 10000, newRateLimitManager(), getListOfClusters(10000)},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				tt.rl.LimitClusterMsg(tt.clustersList[i%tt.clusters])
			}
		})
	}
}
