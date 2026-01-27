package ratelimiter

import (
	"testing"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/rate"
	"github.com/stretchr/testify/require"
)

func TestLimiter_DisabledRateAllowsAll(t *testing.T) {
	ResetLimiterForTest()
	t.Setenv(env.VMIndexReportRateLimit.EnvVar(), "0")
	t.Setenv(env.VMIndexReportBucketCapacity.EnvVar(), "5")

	limiter := Limiter()
	for i := 0; i < 5; i++ {
		allowed, reason := limiter.TryConsume("cluster-1")
		require.True(t, allowed)
		require.Empty(t, reason)
	}
}

func TestLimiter_PositiveRateLimitsAfterBucketCapacity(t *testing.T) {
	ResetLimiterForTest()
	t.Setenv(env.VMIndexReportRateLimit.EnvVar(), "1")
	t.Setenv(env.VMIndexReportBucketCapacity.EnvVar(), "2")

	limiter := Limiter()

	for i := 0; i < 2; i++ {
		allowed, reason := limiter.TryConsume("cluster-1")
		require.True(t, allowed, "request %d should be allowed", i+1)
		require.Empty(t, reason, "request %d should not be rate limited", i+1)
	}

	allowed, reason := limiter.TryConsume("cluster-1")
	require.False(t, allowed, "third immediate request should be rate limited")
	require.Equal(t, rate.ReasonRateLimitExceeded, reason)
}

func TestLimiter_InvalidRateEnvFallsBackAndIsUsable(t *testing.T) {
	ResetLimiterForTest()
	t.Setenv(env.VMIndexReportRateLimit.EnvVar(), "not-a-number")
	t.Setenv(env.VMIndexReportBucketCapacity.EnvVar(), "1")

	limiter := Limiter()

	allowed, reason := limiter.TryConsume("cluster-1")
	require.True(t, allowed, "first request should be allowed with fallback config")
	require.Empty(t, reason, "first request should not be rate limited")

	allowed, reason = limiter.TryConsume("cluster-1")
	require.False(t, allowed, "second immediate request should be rate limited with fallback config")
	require.Equal(t, rate.ReasonRateLimitExceeded, reason)
}

func TestOnClientDisconnect_NoPanic(t *testing.T) {
	ResetLimiterForTest()
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
