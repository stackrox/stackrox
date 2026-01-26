package ratelimiter

import (
	"testing"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/require"
)

func resetLimiterForTest() {
	once = sync.Once{}
	instance = nil
}

func TestLimiter_DisabledRateAllowsAll(t *testing.T) {
	resetLimiterForTest()
	t.Setenv(env.VMIndexReportRateLimit.EnvVar(), "0")
	t.Setenv(env.VMIndexReportBucketCapacity.EnvVar(), "5")

	limiter := Limiter()
	for i := 0; i < 5; i++ {
		allowed, reason := limiter.TryConsume("cluster-1")
		require.True(t, allowed)
		require.Empty(t, reason)
	}
}

func TestOnClientDisconnect_NoPanic(t *testing.T) {
	resetLimiterForTest()
	t.Setenv(env.VMIndexReportRateLimit.EnvVar(), "1")
	t.Setenv(env.VMIndexReportBucketCapacity.EnvVar(), "1")

	limiter := Limiter()
	_, _ = limiter.TryConsume("cluster-1")

	require.NotPanics(t, func() {
		OnClientDisconnect("cluster-1")
	})
	require.NotPanics(t, func() {
		OnClientDisconnect("cluster-missing")
	})
}
