package connection

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRateLimitManagerDefaultMaxInitSync(t *testing.T) {
	m := NewRateLimitManager()
	assert.Equal(t, math.MaxInt, m.maxSensors)

	assert.True(t, m.Add("test-1"), "Can add if limit is set to 0")
	assert.Len(t, m.initSyncSensors, 1)

	m.Remove("test-1")
	assert.Len(t, m.initSyncSensors, 0)
}

func TestNewRateLimitManagerNegativeMaxInitSync(t *testing.T) {
	t.Setenv(env.CentralMaxInitSyncSensors.EnvVar(), "-1")
	assert.Panics(t, func() { NewRateLimitManager() })
}

func TestNewRateLimitManagerZeroMaxInitSync(t *testing.T) {
	t.Setenv(env.CentralMaxInitSyncSensors.EnvVar(), "0")
	m := NewRateLimitManager()
	assert.Equal(t, math.MaxInt, m.maxSensors)

	assert.True(t, m.Add("test-1"), "Can add if limit is set to 0")
	assert.Len(t, m.initSyncSensors, 1)

	m.Remove("test-1")
	assert.Len(t, m.initSyncSensors, 0)
}

func TestNewRateLimitManagerMaxInitSync(t *testing.T) {
	t.Setenv(env.CentralMaxInitSyncSensors.EnvVar(), "3")
	m := NewRateLimitManager()

	for i := 0; i < 3; i++ {
		assert.True(t, m.Add(fmt.Sprintf("test-%d", i)))
	}
	assert.False(t, m.Add("test-a"), "Unable to add after limit is reached")
	assert.Len(t, m.initSyncSensors, 3)

	m.Remove("test-a")
	assert.False(t, m.Add("test-a"), "Unable to add after removing non-existing")

	m.Remove("test-1")
	assert.Len(t, m.initSyncSensors, 2)
	assert.True(t, m.Add("test-a"), "Can add after one is removed")
	assert.Len(t, m.initSyncSensors, 3)

	assert.False(t, m.Add("test-b"), "Unable to add after limit is reached")
}

func TestNewRateLimitManagerDefaultEventsPerSecond(t *testing.T) {
	m := NewRateLimitManager()

	for i := 0; i < 100; i++ {
		require.False(t, m.LimitMsg(), "No limit")
	}
}

func TestNewRateLimitManagerNegativeEventsPerSecond(t *testing.T) {
	t.Setenv(env.CentralSensorMaxEventsPerSecond.EnvVar(), "-1")
	assert.Panics(t, func() { NewRateLimitManager() })
}

func TestNewRateLimitManagerZeroEventsPerSecond(t *testing.T) {
	t.Setenv(env.CentralSensorMaxEventsPerSecond.EnvVar(), "0")
	m := NewRateLimitManager()

	for i := 0; i < 100; i++ {
		require.False(t, m.LimitMsg(), "No limit")
	}
}

func TestNewRateLimitManagerEventsPerSecond(t *testing.T) {
	t.Setenv(env.CentralSensorMaxEventsPerSecond.EnvVar(), "10")
	m := NewRateLimitManager()

	hitLimit := false
	for i := 0; i < 30; i++ {
		limitMsg := m.LimitMsg()
		if i < 10 {
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
	time.Sleep(2 * time.Second)

	assert.False(t, m.LimitMsg(), "Rate is below threshold")
}
