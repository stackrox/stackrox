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
	protoTime := protocompat.ConvertTimeToTimestampOrNil(&createdTime)

	reg.EXPECT().Metadata(gomock.Any()).DoAndReturn(func(img *storage.Image) (*storage.ImageMetadata, error) {
		assert.Equal(t, "v1.0.0", img.GetName().GetTag())
		assert.Equal(t, "registry.example.com", img.GetName().GetRegistry())
		assert.Equal(t, "repo/image", img.GetName().GetRemote())
		return &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Created: protoTime,
			},
			V2: &storage.V2Metadata{
				Digest: "sha256:abc123",
			},
			LayerShas: []string{"sha256:layer1", "sha256:layer2"},
		}, nil
	})

	fetcher := NewTagFetcher(reg, name, testLimiter(), nil)
	collected := collectResults(fetcher.Fetch(context.Background(), []string{"v1.0.0"}))

	require.Len(t, collected, 1)
	assert.Equal(t, "v1.0.0", collected[0].Tag)
	assert.Equal(t, "sha256:abc123", collected[0].ManifestDigest)
	assert.NotNil(t, collected[0].Created)
	assert.Equal(t, createdTime, *collected[0].Created)
	assert.Equal(t, []string{"sha256:layer1", "sha256:layer2"}, collected[0].LayerDigests)
	assert.Nil(t, collected[0].Error)
}

func TestTagFetcher_MultipleTagsConcurrent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	reg := mocks.NewMockRegistry(ctrl)
	name := testImageName("registry.example.com", "repo/image")

	tags := []string{"v1.0.0", "v1.1.0", "v1.2.0"}

	createdTime := time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)
	protoTime := protocompat.ConvertTimeToTimestampOrNil(&createdTime)

	reg.EXPECT().Metadata(gomock.Any()).Times(3).DoAndReturn(func(img *storage.Image) (*storage.ImageMetadata, error) {
		return &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Created: protoTime,
			},
			V2: &storage.V2Metadata{
				Digest: "sha256:" + img.GetName().GetTag(),
			},
		}, nil
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
	protoTime := protocompat.ConvertTimeToTimestampOrNil(&createdTime)

	reg.EXPECT().Metadata(gomock.Any()).DoAndReturn(func(img *storage.Image) (*storage.ImageMetadata, error) {
		if img.GetName().GetTag() == "v1.1.0" {
			return nil, errors.New("registry error")
		}
		return &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Created: protoTime,
			},
			V2: &storage.V2Metadata{
				Digest: "sha256:" + img.GetName().GetTag(),
			},
		}, nil
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
	var callsStarted int32

	createdTime := time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)
	protoTime := protocompat.ConvertTimeToTimestampOrNil(&createdTime)

	// Mock returns with a short delay. Goroutines will be in-flight when we cancel.
	reg.EXPECT().Metadata(gomock.Any()).AnyTimes().DoAndReturn(func(img *storage.Image) (*storage.ImageMetadata, error) {
		atomic.AddInt32(&callsStarted, 1)
		time.Sleep(100 * time.Millisecond)
		return &storage.ImageMetadata{
			V1: &storage.V1Metadata{Created: protoTime},
			V2: &storage.V2Metadata{Digest: "sha256:" + img.GetName().GetTag()},
		}, nil
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
	assert.Greater(t, atomic.LoadInt32(&callsStarted), int32(0), "Expected some metadata calls to start")

	// Results may be partial due to cancellation - that's expected.
	t.Logf("Collected %d results, %d calls started", len(collected), atomic.LoadInt32(&callsStarted))
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
	protoTime := protocompat.ConvertTimeToTimestampOrNil(&createdTime)

	reg.EXPECT().Metadata(gomock.Any()).Times(2).DoAndReturn(func(img *storage.Image) (*storage.ImageMetadata, error) {
		tag := img.GetName().GetTag()
		if tag == "v1.0.0" {
			return &storage.ImageMetadata{
				V1: &storage.V1Metadata{Created: protoTime},
				V2: &storage.V2Metadata{Digest: "sha256:unchanged"}, // Same as known.
			}, nil
		}
		return &storage.ImageMetadata{
			V1: &storage.V1Metadata{Created: protoTime},
			V2: &storage.V2Metadata{Digest: "sha256:new-digest"}, // Different from known.
		}, nil
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

	metadata := &storage.ImageMetadata{
		V1: &storage.V1Metadata{
			Created: protoTime,
		},
		V2: &storage.V2Metadata{
			Digest: "sha256:abc123",
		},
		LayerShas: []string{"sha256:layer1", "sha256:layer2", "sha256:layer3"},
	}

	result := tagMetadata("v1.0.0", metadata)

	assert.Equal(t, "v1.0.0", result.Tag)
	assert.Equal(t, "sha256:abc123", result.ManifestDigest)
	assert.NotNil(t, result.Created)
	assert.Equal(t, createdTime, *result.Created)
	assert.Equal(t, []string{"sha256:layer1", "sha256:layer2", "sha256:layer3"}, result.LayerDigests)
	assert.Nil(t, result.Error)
}

func Test_tagMetadata_MissingV2FallsBackToV1Digest(t *testing.T) {
	createdTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	protoTime := protocompat.ConvertTimeToTimestampOrNil(&createdTime)

	metadata := &storage.ImageMetadata{
		V1: &storage.V1Metadata{
			Digest:  "sha256:v1-digest",
			Created: protoTime,
		},
		// V2 is nil - should fallback to V1 digest
	}

	result := tagMetadata("v1.0.0", metadata)

	assert.Equal(t, "sha256:v1-digest", result.ManifestDigest) // Fallback to V1 digest.
	assert.NotNil(t, result.Created)
	assert.Equal(t, createdTime, *result.Created)
	assert.Empty(t, result.LayerDigests) // No layers in metadata.
	assert.Nil(t, result.Error)
}

func Test_tagMetadata_V2DigestPreferredOverV1(t *testing.T) {
	createdTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	protoTime := protocompat.ConvertTimeToTimestampOrNil(&createdTime)

	metadata := &storage.ImageMetadata{
		V1: &storage.V1Metadata{
			Digest:  "sha256:v1-digest",
			Created: protoTime,
		},
		V2: &storage.V2Metadata{
			Digest: "sha256:v2-digest",
		},
	}

	result := tagMetadata("v1.0.0", metadata)

	// V2 digest should be preferred when both exist.
	assert.Equal(t, "sha256:v2-digest", result.ManifestDigest)
	assert.NotNil(t, result.Created)
	assert.Nil(t, result.Error)
}
