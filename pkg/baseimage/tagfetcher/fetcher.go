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

// fetch fetches metadata at the registry for a single tag.
func (f *TagFetcher) metadata(ctx context.Context, name *storage.ImageName) (*storage.ImageMetadata, error) {
	if err := f.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter wait: %w", err)
	}
	metadata, err := f.registry.Metadata(&storage.Image{Name: name})
	if err != nil {
		return nil, fmt.Errorf("registry call failed: %w", err)
	}
	if metadata == nil {
		return nil, errors.New("nil metadata returned")
	}
	if metadata.GetV1() == nil {
		return nil, errors.New("nil V1 metadata returned")
	}
	if metadata.GetV1().GetCreated() == nil {
		return nil, errors.New("nil V1 created metadata returned")
	}
	return metadata, nil
}

// tagMetadata extracts TagMetadata from storage.ImageMetadata.
//
// TODO(ROX-32382): Manifest lists are not handled currently, and it will require
// a new interface to retrieve metadata from the registry.
func tagMetadata(tag string, metadata *storage.ImageMetadata) TagMetadata {
	result := TagMetadata{Tag: tag}

	// Extract digest from V2 fallback to V1
	result.ManifestDigest = stringutils.FirstNonEmpty(
		metadata.GetV2().GetDigest(),
		metadata.GetV1().GetDigest(),
	)

	// Extract creation timestamp from V1 metadata.
	t := protocompat.ConvertTimestampToTimeOrNil(metadata.GetV1().GetCreated())
	result.Created = t

	// Extract layer digests for base image matching via layer commonality.
	result.LayerDigests = metadata.GetLayerShas()

	return result
}
