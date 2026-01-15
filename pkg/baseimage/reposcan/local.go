package reposcan

import (
	"context"
	"fmt"
	"iter"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/baseimage/tagfetcher"
	"github.com/stackrox/rox/pkg/env"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/time/rate"
)

var log = logging.LoggerForModule()

// LocalScanner scans repositories directly from Central.
type LocalScanner struct {
	registries registries.Set

	// rateLimiters stores per-integration rate limiters. The key is the
	// integration ID. All repositories using the same integration share
	// a rate limiter to avoid exceeding registry quotas.
	rateLimiters   map[string]*rate.Limiter
	rateLimitersMu sync.Mutex
}

// NewLocalScanner creates a LocalScanner with the given registries.
func NewLocalScanner(registries registries.Set) *LocalScanner {
	return &LocalScanner{
		registries:   registries,
		rateLimiters: make(map[string]*rate.Limiter),
	}
}

// Name implements Scanner.
func (c *LocalScanner) Name() string {
	return "local"
}

// ScanRepository implements Scanner.
func (c *LocalScanner) ScanRepository(
	ctx context.Context,
	repo *storage.BaseImageRepository,
	req ScanRequest,
) iter.Seq2[TagEvent, error] {
	return func(yield func(TagEvent, error) bool) {
		// Parse repository path.
		name, _, err := imageUtils.GenerateImageNameFromString(repo.GetRepositoryPath())
		if err != nil {
			yield(TagEvent{}, fmt.Errorf("parsing repository path %q: %w", repo.GetRepositoryPath(), err))
			return
		}

		// Find matching registry integration.
		reg := c.findRegistry(name)
		if reg == nil {
			yield(TagEvent{}, fmt.Errorf("no matching image integration found for repository %s", repo.GetRepositoryPath()))
			return
		}

		// List and filter tags.
		tags, err := ListAndFilterTags(ctx, reg, name.GetRemote(), req.Pattern)
		if err != nil {
			yield(TagEvent{}, fmt.Errorf("listing tags for repository %s: %w", repo.GetRepositoryPath(), err))
			return
		}

		// Track tags that were seen in the registry and that need metadata fetch.
		seen := make(map[string]bool)
		toFetch := make([]string, 0, len(tags))
		for _, tag := range tags {
			if seen[tag] {
				log.Infof("Skipping duplicate tag %s in repository %s", tag, repo.GetRepositoryPath())
				continue
			}
			seen[tag] = true
			if _, ok := req.SkipTags[tag]; ok {
				// Skip fetching.
				continue
			}
			toFetch = append(toFetch, tag)
		}

		// Emit deletion events as soon as possible, leaving potential stream
		// failures to the end.
		for tag := range req.CheckTags {
			if !seen[tag] {
				if !yield(TagEvent{Tag: tag, Type: TagEventDeleted}, nil) {
					return
				}
			}
		}
		for tag := range req.SkipTags {
			if !seen[tag] {
				if !yield(TagEvent{Tag: tag, Type: TagEventDeleted}, nil) {
					return
				}
			}
		}

		// Fetch metadata concurrently for tags that need it.
		if len(toFetch) > 0 {
			// Create fetcher for this repo, uses a list of cached digests and a shared the
			// limiter per registry integration.
			id := reg.Source().GetId()
			limiter := c.getRateLimiter(id)
			digests := make(map[string]string, len(req.CheckTags))
			for tag, tagInfo := range req.CheckTags {
				digests[tag] = tagInfo.GetManifestDigest()
			}
			fetcher := tagfetcher.NewTagFetcher(reg, name, limiter, digests)

			// Fetch tags, and yield events.
			for metadata := range fetcher.Fetch(ctx, toFetch) {
				var event TagEvent
				if metadata.Error != nil {
					event = TagEvent{
						Tag:   metadata.Tag,
						Type:  TagEventError,
						Error: metadata.Error,
					}
				} else {
					event = TagEvent{
						Tag:      metadata.Tag,
						Type:     TagEventMetadata,
						Metadata: &metadata,
					}
				}

				if !yield(event, nil) {
					return
				}
			}
		}
	}
}

func (c *LocalScanner) findRegistry(name *storage.ImageName) types.ImageRegistry {
	var regs []types.ImageRegistry
	if env.DedupeImageIntegrations.BooleanSetting() {
		regs = c.registries.GetAllUnique()
	} else {
		regs = c.registries.GetAll()
	}
	for _, r := range regs {
		if r.Match(name) {
			return r
		}
	}
	return nil
}

// getRateLimiter returns the rate limiter for the given integration.
// If one doesn't exist, it creates a new one using the configured rate limit.
func (c *LocalScanner) getRateLimiter(integrationID string) *rate.Limiter {
	c.rateLimitersMu.Lock()
	defer c.rateLimitersMu.Unlock()

	if limiter, ok := c.rateLimiters[integrationID]; ok {
		return limiter
	}

	rateLimit := env.BaseImageWatcherRegistryRateLimit.IntegerSetting()
	limiter := rate.NewLimiter(rate.Limit(rateLimit), 1)
	c.rateLimiters[integrationID] = limiter
	return limiter
}
