//go:build integration

package watcher

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/baseimage/tagfetcher"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

// TestMetadataFetcher_DockerHub_Integration tests metadata fetching from Docker Hub.
// This test requires network access to Docker Hub.
func TestMetadataFetcher_DockerHub_Integration(t *testing.T) {
	// Create a Docker Hub registry client (unauthenticated).
	integration := &storage.ImageIntegration{
		Name: "Docker Hub Integration Test",
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: "https://registry-1.docker.io",
			},
		},
	}

	reg, err := docker.NewDockerRegistry(integration, true, nil)
	require.NoError(t, err, "Failed to create Docker registry client")

	// Test with alpine:3.18 and alpine:3.19 - well-known public images.
	tags := []string{"3.18", "3.19"}
	imageName := &storage.ImageName{
		Registry: "docker.io",
		Remote:   "library/alpine",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Use a conservative rate limiter for public registry.
	limiter := rate.NewLimiter(rate.Limit(5.0), 1)
	fetcher := tagfetcher.NewTagFetcher(reg, imageName, limiter, nil)

	var collected []tagfetcher.TagMetadata
	for r := range fetcher.Fetch(ctx, tags) {
		collected = append(collected, r)
	}

	require.Len(t, collected, 2)

	var successCount int
	for _, result := range collected {
		t.Logf("Tag: %s", result.Tag)
		t.Logf("  Digest: %s", result.ManifestDigest)
		if result.Created != nil {
			t.Logf("  Created: %s", result.Created.Format(time.RFC3339))
		}
		t.Logf("  Layers: %d", len(result.LayerDigests))
		if result.Error != nil {
			t.Logf("  Error: %v", result.Error)
			continue
		}

		successCount++
		assert.NotEmpty(t, result.ManifestDigest, "Expected manifest digest for tag %s", result.Tag)
		assert.NotNil(t, result.Created, "Expected creation timestamp for tag %s", result.Tag)
		assert.NotEmpty(t, result.LayerDigests, "Expected layer digests for tag %s", result.Tag)
	}

	require.Greater(t, successCount, 0, "Expected at least one successful metadata fetch")
}

// TestMetadataFetcher_RateLimiting_Integration verifies rate limiting behavior.
func TestMetadataFetcher_RateLimiting_Integration(t *testing.T) {
	integration := &storage.ImageIntegration{
		Name: "Docker Hub Integration Test",
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: "https://registry-1.docker.io",
			},
		},
	}

	reg, err := docker.NewDockerRegistry(integration, true, nil)
	require.NoError(t, err, "Failed to create Docker registry client")

	// Use 5 tags with a rate limit of 2/second to test rate limiting.
	tags := []string{"3.16", "3.17", "3.18", "3.19", "3.20"}
	imageName := &storage.ImageName{
		Registry: "docker.io",
		Remote:   "library/alpine",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Rate limit of 2/second for predictable timing.
	limiter := rate.NewLimiter(rate.Limit(2.0), 1)
	fetcher := tagfetcher.NewTagFetcher(reg, imageName, limiter, nil)

	start := time.Now()
	var collected []tagfetcher.TagMetadata
	for r := range fetcher.Fetch(ctx, tags) {
		collected = append(collected, r)
	}
	elapsed := time.Since(start)

	require.Len(t, collected, 5)

	// Verify at least some requests succeeded.
	var successCount int
	for _, result := range collected {
		if result.Error == nil {
			successCount++
		} else {
			t.Logf("Tag %s error: %v", result.Tag, result.Error)
		}
	}
	require.Greater(t, successCount, 0, "Expected at least one successful metadata fetch")

	// With rate limit of 2/second and 5 tags, minimum time should be ~2 seconds.
	// (Actually it's 4 waits of 0.5s each = 2s minimum plus request time.)
	t.Logf("Elapsed time for 5 tags with rate limit 2/s: %v", elapsed)

	// The elapsed time should be at least 1.5 seconds (accounting for some slack).
	assert.True(t, elapsed >= 1500*time.Millisecond,
		"Expected at least 1.5s with rate limiting, got %v", elapsed)
}
