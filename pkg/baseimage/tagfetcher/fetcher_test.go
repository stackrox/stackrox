package tagfetcher

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/registries/types/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/time/rate"
)

func testImageName(registry, remote string) *storage.ImageName {
	return &storage.ImageName{
		Registry: registry,
		Remote:   remote,
	}
}

// testLimiter returns a rate limiter for testing with high throughput.
func testLimiter() *rate.Limiter {
	return rate.NewLimiter(rate.Inf, 1)
}

// collectResults drains results from a channel into a slice.
func collectResults(ch <-chan TagMetadata) []TagMetadata {
	var results []TagMetadata
	for r := range ch {
		results = append(results, r)
	}
	return results
}

// singlePlatform builds an AllPlatformsMetadata for a single linux/amd64 image.
func singlePlatform(digest string, created *time.Time, layers ...string) *types.AllPlatformsMetadata {
	return &types.AllPlatformsMetadata{
		ManifestDigest: digest,
		IsManifestList: false,
		Platforms: []*types.PlatformMetadata{{
			OS:           "linux",
			Architecture: "amd64",
			Metadata: &storage.ImageMetadata{
				V1:        &storage.V1Metadata{Created: protocompat.ConvertTimeToTimestampOrNil(created)},
				V2:        &storage.V2Metadata{Digest: digest},
				LayerShas: layers,
			},
		}},
	}
}

func TestTagFetcher_EmptyTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	reg := mocks.NewMockRegistry(ctrl)
	name := testImageName("registry.example.com", "repo/image")

	fetcher := NewTagFetcher(reg, name, testLimiter(), nil)

	results := collectResults(fetcher.Fetch(context.Background(), nil))
	assert.Empty(t, results)

	results = collectResults(fetcher.Fetch(context.Background(), []string{}))
	assert.Empty(t, results)
}

func TestTagFetcher_SingleTag(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	reg := mocks.NewMockRegistry(ctrl)
	name := testImageName("registry.example.com", "repo/image")

	createdTime := time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)

	reg.EXPECT().MetadataForAllPlatforms(gomock.Any()).DoAndReturn(func(img *storage.Image) (*types.AllPlatformsMetadata, error) {
		assert.Equal(t, "v1.0.0", img.GetName().GetTag())
		assert.Equal(t, "registry.example.com", img.GetName().GetRegistry())
		assert.Equal(t, "repo/image", img.GetName().GetRemote())
		return singlePlatform("sha256:abc123", &createdTime, "sha256:layer1", "sha256:layer2"), nil
	})

	fetcher := NewTagFetcher(reg, name, testLimiter(), nil)
	collected := collectResults(fetcher.Fetch(context.Background(), []string{"v1.0.0"}))

	require.Len(t, collected, 1)
	got := collected[0]
	assert.Equal(t, "v1.0.0", got.Tag)
	assert.Equal(t, "sha256:abc123", got.ManifestDigest)
	assert.False(t, got.IsManifestList)
	assert.NotNil(t, got.Created)
	assert.Equal(t, createdTime, *got.Created)
	assert.Equal(t, []string{"sha256:layer1", "sha256:layer2"}, got.LayerDigests)
	require.Len(t, got.Platforms, 1)
	assert.Equal(t, "linux", got.Platforms[0].OS)
	assert.Equal(t, "amd64", got.Platforms[0].Architecture)
	assert.Equal(t, []string{"sha256:layer1", "sha256:layer2"}, got.Platforms[0].LayerDigests)
	assert.Equal(t, map[string]string{"linux/amd64": "sha256:abc123"}, got.ListDigests)
	assert.Nil(t, got.Error)
}

func TestTagFetcher_MultipleTagsConcurrent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	reg := mocks.NewMockRegistry(ctrl)
	name := testImageName("registry.example.com", "repo/image")

	tags := []string{"v1.0.0", "v1.1.0", "v1.2.0"}

	createdTime := time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)

	reg.EXPECT().MetadataForAllPlatforms(gomock.Any()).Times(3).DoAndReturn(func(img *storage.Image) (*types.AllPlatformsMetadata, error) {
		return singlePlatform("sha256:"+img.GetName().GetTag(), &createdTime), nil
	})

	fetcher := NewTagFetcher(reg, name, testLimiter(), nil)
	collected := collectResults(fetcher.Fetch(context.Background(), tags))

	require.Len(t, collected, 3)

	// Results may not be in order, so check by tag.
	tagToDigest := make(map[string]string)
	for _, r := range collected {
		tagToDigest[r.Tag] = r.ManifestDigest
		assert.Nil(t, r.Error)
	}
	for _, tag := range tags {
		assert.Equal(t, "sha256:"+tag, tagToDigest[tag])
	}
}

func TestTagFetcher_WithErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	reg := mocks.NewMockRegistry(ctrl)
	name := testImageName("registry.example.com", "repo/image")

	createdTime := time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)

	reg.EXPECT().MetadataForAllPlatforms(gomock.Any()).DoAndReturn(func(img *storage.Image) (*types.AllPlatformsMetadata, error) {
		if img.GetName().GetTag() == "v1.1.0" {
			return nil, errors.New("registry error")
		}
		return singlePlatform("sha256:"+img.GetName().GetTag(), &createdTime), nil
	}).Times(2)

	fetcher := NewTagFetcher(reg, name, testLimiter(), nil)
	collected := collectResults(fetcher.Fetch(context.Background(), []string{"v1.0.0", "v1.1.0"}))

	require.Len(t, collected, 2)

	// Find results by tag.
	var successResult, errorResult TagMetadata
	for _, r := range collected {
		if r.Tag == "v1.0.0" {
			successResult = r
		} else {
			errorResult = r
		}
	}

	assert.Nil(t, successResult.Error)
	assert.NotNil(t, errorResult.Error)
	assert.Contains(t, errorResult.Error.Error(), "registry error")
}

func TestTagFetcher_ContextCancellation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	reg := mocks.NewMockRegistry(ctrl)
	name := testImageName("registry.example.com", "repo/image")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	tags := []string{"v1.0.0", "v1.1.0"}
	fetcher := NewTagFetcher(reg, name, testLimiter(), nil)
	collected := collectResults(fetcher.Fetch(ctx, tags))

	// With cancelled context, function returns early without processing.
	// May have 0-2 results depending on timing; any results should have errors.
	for _, r := range collected {
		assert.NotNil(t, r.Error)
	}
}

// TestTagFetcher_ContextCancellationDuringFetch verifies that cancelling the
// context while goroutines are actively fetching does not cause a panic
// (send on closed channel). This tests the fix where defer wg.Wait() ensures
// all goroutines complete before the channel is closed.
func TestTagFetcher_ContextCancellationDuringFetch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	reg := mocks.NewMockRegistry(ctrl)
	name := testImageName("registry.example.com", "repo/image")

	// Create many tags to increase chance of race condition.
	tags := make([]string, 20)
	for i := range tags {
		tags[i] = fmt.Sprintf("v1.%d.0", i)
	}

	// Track how many metadata calls started.
	var callsStarted atomic.Int32

	createdTime := time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)

	// Mock returns with a short delay. Goroutines will be in-flight when we cancel.
	reg.EXPECT().MetadataForAllPlatforms(gomock.Any()).AnyTimes().DoAndReturn(func(img *storage.Image) (*types.AllPlatformsMetadata, error) {
		callsStarted.Add(1)
		time.Sleep(100 * time.Millisecond)
		return singlePlatform("sha256:"+img.GetName().GetTag(), &createdTime), nil
	})

	ctx, cancel := context.WithCancel(context.Background())

	// Start fetching.
	resultsCh := NewTagFetcher(reg, name, testLimiter(), nil).Fetch(ctx, tags)

	// Wait for goroutines to start, then cancel. Without the fix (defer wg.Wait()),
	// the channel would close while goroutines are still trying to send, causing panic.
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Drain results - if the bug exists, this would panic with "send on closed channel".
	var collected []TagMetadata
	for r := range resultsCh {
		collected = append(collected, r)
	}

	// Verify at least some calls started before cancellation.
	assert.Greater(t, callsStarted.Load(), int32(0), "Expected some metadata calls to start")

	// Results may be partial due to cancellation - that's expected.
	t.Logf("Collected %d results, %d calls started", len(collected), callsStarted.Load())
}

func TestTagFetcher_SkipsUnchangedDigests(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	reg := mocks.NewMockRegistry(ctrl)
	name := testImageName("registry.example.com", "repo/image")

	// v1.0.0 has unchanged digest, v1.1.0 has changed digest.
	knownDigests := map[string]string{
		"v1.0.0": "sha256:unchanged",
		"v1.1.0": "sha256:old-digest",
	}

	createdTime := time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)

	reg.EXPECT().MetadataForAllPlatforms(gomock.Any()).Times(2).DoAndReturn(func(img *storage.Image) (*types.AllPlatformsMetadata, error) {
		tag := img.GetName().GetTag()
		if tag == "v1.0.0" {
			return singlePlatform("sha256:unchanged", &createdTime), nil // Same as known.
		}
		return singlePlatform("sha256:new-digest", &createdTime), nil // Different from known.
	})

	fetcher := NewTagFetcher(reg, name, testLimiter(), knownDigests)
	collected := collectResults(fetcher.Fetch(context.Background(), []string{"v1.0.0", "v1.1.0"}))

	// Only v1.1.0 should be in results (v1.0.0 skipped due to unchanged digest).
	require.Len(t, collected, 1)
	assert.Equal(t, "v1.1.0", collected[0].Tag)
	assert.Equal(t, "sha256:new-digest", collected[0].ManifestDigest)
}

func Test_tagMetadata_FullMetadata(t *testing.T) {
	createdTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	protoTime := protocompat.ConvertTimeToTimestampOrNil(&createdTime)

	metadata := &types.AllPlatformsMetadata{
		ManifestDigest: "sha256:abc123",
		Platforms: []*types.PlatformMetadata{{
			OS:           "linux",
			Architecture: "amd64",
			Metadata: &storage.ImageMetadata{
				V1: &storage.V1Metadata{
					Created: protoTime,
					Layers: []*storage.ImageLayer{
						{Instruction: "ADD", Value: "rootfs"},
						{Instruction: "USER", Value: "1001", Empty: true},
					},
				},
				V2:        &storage.V2Metadata{Digest: "sha256:abc123"},
				LayerShas: []string{"sha256:layer1", "sha256:layer2", "sha256:layer3"},
			},
		}},
	}

	result := tagMetadata("v1.0.0", metadata)

	assert.Equal(t, "v1.0.0", result.Tag)
	assert.Equal(t, "sha256:abc123", result.ManifestDigest)
	assert.NotNil(t, result.Created)
	assert.Equal(t, createdTime, *result.Created)
	assert.Equal(t, []string{"sha256:layer1", "sha256:layer2", "sha256:layer3"}, result.LayerDigests)
	require.Len(t, result.History, 2)
	assert.Equal(t, "ADD", result.History[0].GetInstruction())
	assert.Equal(t, "rootfs", result.History[0].GetValue())
	assert.False(t, result.History[0].GetEmpty())
	assert.Equal(t, "USER", result.History[1].GetInstruction())
	assert.Equal(t, "1001", result.History[1].GetValue())
	assert.True(t, result.History[1].GetEmpty())

	require.Len(t, result.Platforms, 1)
	assert.Equal(t, []string{"sha256:layer1", "sha256:layer2", "sha256:layer3"}, result.Platforms[0].LayerDigests)
	assert.Equal(t, map[string]string{"linux/amd64": "sha256:abc123"}, result.ListDigests)
	assert.Nil(t, result.Error)
}

func Test_tagMetadata_MultiPlatformDistinguishesVariants(t *testing.T) {
	createdTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	protoTime := protocompat.ConvertTimeToTimestampOrNil(&createdTime)

	platform := func(os, arch, variant, digest string, layers ...string) *types.PlatformMetadata {
		return &types.PlatformMetadata{
			OS:           os,
			Architecture: arch,
			Variant:      variant,
			Metadata: &storage.ImageMetadata{
				V1:        &storage.V1Metadata{Created: protoTime},
				V2:        &storage.V2Metadata{Digest: digest},
				LayerShas: layers,
			},
		}
	}

	metadata := &types.AllPlatformsMetadata{
		ManifestDigest: "sha256:list",
		IsManifestList: true,
		Platforms: []*types.PlatformMetadata{
			platform("linux", "amd64", "", "sha256:amd", "sha256:amd-l1"),
			platform("linux", "arm", "v6", "sha256:armv6", "sha256:armv6-l1"),
			platform("linux", "arm", "v7", "sha256:armv7", "sha256:armv7-l1"),
		},
	}

	result := tagMetadata("8.10", metadata)

	assert.Equal(t, "sha256:list", result.ManifestDigest)
	assert.True(t, result.IsManifestList)
	require.Len(t, result.Platforms, 3)

	// Variants must not collide in ListDigests.
	assert.Equal(t, map[string]string{
		"linux/amd64":  "sha256:amd",
		"linux/arm/v6": "sha256:armv6",
		"linux/arm/v7": "sha256:armv7",
	}, result.ListDigests)

	// Primary platform (first) feeds the top-level fields.
	assert.Equal(t, []string{"sha256:amd-l1"}, result.LayerDigests)
}

func Test_tagMetadata_MissingTopLevelDigestFallsBackToPlatform(t *testing.T) {
	createdTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	protoTime := protocompat.ConvertTimeToTimestampOrNil(&createdTime)

	metadata := &types.AllPlatformsMetadata{
		ManifestDigest: "", // registry did not report a top-level digest
		Platforms: []*types.PlatformMetadata{{
			OS:           "linux",
			Architecture: "amd64",
			Metadata: &storage.ImageMetadata{
				V1: &storage.V1Metadata{Digest: "sha256:v1-digest", Created: protoTime},
				// V2 nil - platform digest falls back to V1 digest.
			},
		}},
	}

	result := tagMetadata("v1.0.0", metadata)

	assert.Equal(t, "sha256:v1-digest", result.ManifestDigest)
	assert.NotNil(t, result.Created)
	assert.Empty(t, result.LayerDigests)
	assert.Nil(t, result.Error)
}
