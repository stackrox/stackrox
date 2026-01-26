package ratelimiter

import (
	"testing"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stretchr/testify/require"
)

func TestLimiter_DisabledRateAllowsAll(t *testing.T) {
	t.Setenv(env.VMIndexReportRateLimit.EnvVar(), "0")
	t.Setenv(env.VMIndexReportBucketCapacity.EnvVar(), "5")

	limiter := NewFromEnv()
	for i := 0; i < 5; i++ {
		allowed, reason := limiter.TryConsume("cluster-1")
		require.True(t, allowed)
		require.Empty(t, reason)
	}
}

func TestOnClientDisconnect_NoPanic(t *testing.T) {
	t.Setenv(env.VMIndexReportRateLimit.EnvVar(), "1")
	t.Setenv(env.VMIndexReportBucketCapacity.EnvVar(), "1")

	limiter := NewFromEnv()
	_, _ = limiter.TryConsume("cluster-1")

	require.NotPanics(t, func() {
		limiter.OnClientDisconnect("cluster-1")
	})
	require.NotPanics(t, func() {
		limiter.OnClientDisconnect("cluster-missing")
	})
}
