package tagfetcher

import (
	"context"
	"errors"
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
)

// TagFetcher fetches metadata for image tags with rate limiting and concurrency control.
// Each instance is designed for a single scan operation and should not be reused.
type TagFetcher struct {
	registry  types.Registry
	imageName *storage.ImageName
	limiter   *rate.Limiter

	// knownDigests maps tag names to their known manifest digests.
	// Tags with unchanged digests are skipped (not sent to results channel).
	knownDigests map[string]string

	// maxConcurrency is the maximum number of concurrent requests per metadata call.
	maxConcurrency int
}

// NewTagFetcher creates a TagFetcher for fetching tag metadata from a registry.
// The limiter parameter allows sharing rate limits across multiple fetchers.
// KnownDigests maps tag to their last known manifest digests. During fetching,
// tags with unmodified digest are skipped.
func NewTagFetcher(
	registry types.Registry,
	imageName *storage.ImageName,
	limiter *rate.Limiter,
	knownDigests map[string]string,
) *TagFetcher {
	return &TagFetcher{
		registry:       registry,
		imageName:      imageName,
		limiter:        limiter,
		knownDigests:   knownDigests,
		maxConcurrency: 5,
	}
}

// Fetch fetches metadata for the given tags and sends results to the returned channel.
// The channel is closed when all fetches complete.
// Tags with unchanged digests (matching knownDigests) are not sent to the channel.
func (f *TagFetcher) Fetch(ctx context.Context, tags []string) <-chan TagMetadata {
	results := make(chan TagMetadata, f.maxConcurrency)

	go func() {
		defer close(results)
		f.fetchTags(ctx, tags, results)
	}()

	return results
}

// fetchTags performs the concurrent fetch operation.
func (f *TagFetcher) fetchTags(ctx context.Context, tags []string, results chan<- TagMetadata) {
	// Nothing to do.
	if len(tags) == 0 {
		return
	}

	sem := semaphore.NewWeighted(int64(f.maxConcurrency))
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	for _, tag := range tags {
		// Check for context cancellation before starting work.
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Build a new image name for this tag.
		tagName := f.imageName.CloneVT()
		imageUtils.SetImageTagNoSha(tagName, tag)

		wg.Add(1)
		if err := sem.Acquire(ctx, 1); err != nil {
			results <- TagMetadata{
				Tag:   tag,
				Error: err,
			}
			wg.Done()
			continue
		}

		go func(name *storage.ImageName, digest string) {
			defer sem.Release(1)
			defer wg.Done()

			m, err := f.metadata(ctx, name)
			if err != nil {
				results <- TagMetadata{
					Tag:   name.GetTag(),
					Error: fmt.Errorf("fetching metadata for tag %q: %w", name.GetTag(), err),
				}
				return
			}

			tm := tagMetadata(name.GetTag(), m)
			if digest != "" && tm.ManifestDigest == digest {
				// Skip since digest hasn't changed
				return
			}
			results <- tm
		}(tagName, f.knownDigests[tag])
	}
}

// metadata fetches per-platform metadata at the registry for a single tag.
func (f *TagFetcher) metadata(ctx context.Context, name *storage.ImageName) (*types.AllPlatformsMetadata, error) {
	if err := f.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter wait: %w", err)
	}
	metadata, err := f.registry.MetadataForAllPlatforms(&storage.Image{Name: name})
	if err != nil {
		return nil, fmt.Errorf("registry call failed: %w", err)
	}
	if metadata == nil || len(metadata.Platforms) == 0 {
		return nil, errors.New("no platform metadata returned")
	}
	for _, p := range metadata.Platforms {
		if p.Metadata.GetV1() == nil {
			return nil, errors.New("nil V1 metadata returned")
		}
		if p.Metadata.GetV1().GetCreated() == nil {
			return nil, errors.New("nil V1 created metadata returned")
		}
	}
	return metadata, nil
}

// tagMetadata extracts TagMetadata from per-platform registry metadata.
func tagMetadata(tag string, metadata *types.AllPlatformsMetadata) TagMetadata {
	result := TagMetadata{
		Tag:            tag,
		ManifestDigest: metadata.ManifestDigest,
		IsManifestList: metadata.IsManifestList,
		ListDigests:    make(map[string]string, len(metadata.Platforms)),
		Platforms:      make([]PlatformTagMetadata, 0, len(metadata.Platforms)),
	}

	for _, p := range metadata.Platforms {
		// Extract this platform's manifest digest from V2, falling back to V1.
		digest := stringutils.FirstNonEmpty(p.Metadata.GetV2().GetDigest(), p.Metadata.GetV1().GetDigest())
		result.Platforms = append(result.Platforms, PlatformTagMetadata{
			OS:             p.OS,
			Architecture:   p.Architecture,
			Variant:        p.Variant,
			ManifestDigest: digest,
			Created:        protocompat.ConvertTimestampToTimeOrNil(p.Metadata.GetV1().GetCreated()),
			LayerDigests:   p.Metadata.GetLayerShas(),
			History:        historyFromLayers(p.Metadata.GetV1().GetLayers()),
		})
		result.ListDigests[platformKey(p.OS, p.Architecture, p.Variant)] = digest
	}

	// Fall back to the primary platform's digest when the registry did not report a
	// top-level manifest digest (e.g. a registry that omits Docker-Content-Digest).
	if result.ManifestDigest == "" {
		result.ManifestDigest = result.Platforms[0].ManifestDigest
	}

	// The primary platform (first) feeds the watcher's current single-image path until
	// promotion becomes per-platform.
	primary := result.Platforms[0]
	result.Created = primary.Created
	result.LayerDigests = primary.LayerDigests
	result.History = primary.History

	return result
}

// platformKey builds the ListDigests map key: "os/arch", extended with "/variant"
// when a variant is present so that platforms sharing an os/arch do not collide.
func platformKey(os, arch, variant string) string {
	key := os + "/" + arch
	if variant != "" {
		key += "/" + variant
	}
	return key
}

// historyFromLayers copies config history, including empty lines.
// Those lines are already in the fetched V1 metadata. Only digests used to be kept.
func historyFromLayers(layers []*storage.ImageLayer) []*storage.BaseImageHistoryEntry {
	history := make([]*storage.BaseImageHistoryEntry, len(layers))
	for i, layer := range layers {
		history[i] = &storage.BaseImageHistoryEntry{
			Instruction: layer.GetInstruction(),
			Value:       layer.GetValue(),
			Empty:       layer.GetEmpty(),
		}
	}
	return history
}
