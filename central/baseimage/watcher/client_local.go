package watcher

import (
	"context"
	"fmt"
	"iter"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/registries/types"
)

// LocalRepositoryClient scans repositories directly from Central.
type LocalRepositoryClient struct {
	registries registries.Set
}

// NewLocalRepositoryClient creates a LocalRepositoryClient.
func NewLocalRepositoryClient(registries registries.Set) *LocalRepositoryClient {
	return &LocalRepositoryClient{registries: registries}
}

// Name implements RepositoryClient.
func (c *LocalRepositoryClient) Name() string {
	return "local"
}

// ScanRepository implements RepositoryClient.
func (c *LocalRepositoryClient) ScanRepository(
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
		tags, err := listAndFilterTags(ctx, reg, name.GetRemote(), req.Pattern)
		if err != nil {
			yield(TagEvent{}, fmt.Errorf("listing tags for repository %s: %w", repo.GetRepositoryPath(), err))
			return
		}

		// Yield an event for each matching tag.
		for _, tag := range tags {
			if !yield(TagEvent{Tag: tag}, nil) {
				return
			}
		}
	}
}

func (c *LocalRepositoryClient) findRegistry(name *storage.ImageName) types.Registry {
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
