package connection

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stackrox/rox/central/sensor/service/connection/ratetracker"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/ratelimit"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRateLimitManagerDefaultMaxInitSync(t *testing.T) {
	m := newRateLimitManager()
	assert.Equal(t, math.MaxInt, m.initSyncMaxSensors)

	assert.True(t, m.addInitSync("test-1"), "Can add if limit is set to 0")
	assert.Len(t, m.initSyncSensors, 1)

	m.removeCluster("test-1")
	assert.Len(t, m.initSyncSensors, 0)
}

func TestNewRateLimitManagerNegativeMaxInitSync(t *testing.T) {
	t.Setenv(env.CentralMaxInitSyncSensors.EnvVar(), "-1")
	assert.Panics(t, func() { newRateLimitManager() })
}

func TestNewRateLimitManagerZeroMaxInitSync(t *testing.T) {
	t.Setenv(env.CentralMaxInitSyncSensors.EnvVar(), "0")
	m := newRateLimitManager()
	assert.Equal(t, math.MaxInt, m.initSyncMaxSensors)

	assert.True(t, m.addInitSync("test-1"), "Can add if limit is set to 0")
	assert.Len(t, m.initSyncSensors, 1)

	m.removeCluster("test-1")
	assert.Len(t, m.initSyncSensors, 0)
}

func TestNewRateLimitManagerMaxInitSync(t *testing.T) {
	centralMaxInitSyncSensors := 5

	t.Setenv(env.CentralMaxInitSyncSensors.EnvVar(), fmt.Sprintf("%d", centralMaxInitSyncSensors))
	m := newRateLimitManager()

	for i := 0; i < centralMaxInitSyncSensors; i++ {
		assert.True(t, m.addInitSync(fmt.Sprintf("test-%d", i)))
	}
	assert.False(t, m.addInitSync("test-a"), "Unable to add after limit is reached")
	assert.Len(t, m.initSyncSensors, centralMaxInitSyncSensors)

	m.removeCluster("test-a")
	assert.False(t, m.addInitSync("test-a"), "Unable to add after removing non-existing")

	m.removeCluster("test-1")
	assert.Len(t, m.initSyncSensors, centralMaxInitSyncSensors-1)
	assert.True(t, m.addInitSync("test-a"), "Can add after one is removed")
	assert.Len(t, m.initSyncSensors, centralMaxInitSyncSensors)

	assert.False(t, m.addInitSync("test-b"), "Unable to add after limit is reached")
}

func TestRateLimitManagerNilGuards(t *testing.T) {
	tests := []struct {
		name    string
		manager *rateLimitManager
	}{
		{"nil manager", nil},
		{"empty manager", &rateLimitManager{initSyncMaxSensors: 1}},
		{"no tracker", &rateLimitManager{initSyncMaxSensors: 1, msgRateLimiter: ratelimit.NewRateLimiter(10, time.Minute)}},
		{"no rate limiter", &rateLimitManager{initSyncMaxSensors: 1, msgRateTracker: ratetracker.NewClusterRateTracker(time.Minute, 1)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, tt.manager.addInitSync("test-1"))
			assert.False(t, tt.manager.limitClusterMsg("test-1"))
			assert.False(t, tt.manager.throttleMsg("test-1"))
			assert.NotPanics(t, func() { tt.manager.removeCluster("test-1") })
		})
	}
}

func TestNewRateLimitManagerEventsPerSecond(t *testing.T) {
	t.Setenv(env.CentralSensorMaxEventsPerSecond.EnvVar(), "3")
	t.Setenv(env.CentralSensorMaxEventsThrottleDuration.EnvVar(), "0")
	m := newRateLimitManager()

	hitLimit := false
	for i := 0; i < 30; i++ {
		limitMsg := m.limitClusterMsg("c1")
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

	assert.False(t, m.limitClusterMsg("c1"), "Rate is below threshold")
}

func TestNewRateLimitManagerEventsPerSecondWithThrottle(t *testing.T) {
	t.Setenv(env.CentralSensorMaxEventsPerSecond.EnvVar(), "3")
	t.Setenv(env.CentralSensorMaxEventsThrottleDuration.EnvVar(), "1s")
	m := newRateLimitManager()

	var wg sync.WaitGroup

	numCalls := 15
	resultChan := make(chan bool, numCalls)

	for i := 0; i < numCalls; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resultChan <- m.limitClusterMsg("c1")
		}()
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	countLimitHit := 0
	for result := range resultChan {
		if result {
			countLimitHit++
		}
	}

	assert.LessOrEqual(t, countLimitHit, numCalls-3, "Burst messages are not limited")
	assert.GreaterOrEqual(t, numCalls-countLimitHit, 2*3, "Burst and throttled messages are successfully handled")

	// Wait for rate limit to refill.
	time.Sleep(time.Second)

	assert.False(t, m.limitClusterMsg("c1"), "Rate is below threshold")
}

/*** Benchmark tests ***/

func getListOfClusters(n int) []string {
	clusters := make([]string, 0, n)
	for i := 0; i < n; i++ {
		clusters = append(clusters, fmt.Sprintf("cluster-%d", i))
	}

	return clusters
}

func BenchmarkLimitClusterMsg(b *testing.B) {
	tests := []struct {
		name         string
		clusters     int
		rl           *rateLimitManager
		clustersList []string
	}{
		{"00001 clusters", 1, newRateLimitManager(), getListOfClusters(1)},
		{"00010 clusters", 10, newRateLimitManager(), getListOfClusters(10)},
		{"00100 clusters", 100, newRateLimitManager(), getListOfClusters(100)},
		{"01000 clusters", 1000, newRateLimitManager(), getListOfClusters(1000)},
		{"10000 clusters", 10000, newRateLimitManager(), getListOfClusters(10000)},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				tt.rl.limitClusterMsg(tt.clustersList[i%tt.clusters])
			}
		})
	}
}
