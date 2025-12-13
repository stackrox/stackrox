// Package watcher provides base image repository watching functionality for tag discovery.
package watcher

import (
	"context"
	"fmt"
	"path"

	"github.com/stackrox/rox/pkg/registries/types"
)

// listAndFilterTags lists all tags for the given image and filters them using
// the [path.Match] glob pattern. It returns the list of matching tags or a nil
// slice if no tags match the pattern. Empty pattern matches no tag.
func listAndFilterTags(ctx context.Context, registry types.Registry, repo string, pattern string) ([]string, error) {
	allTags, err := registry.ListTags(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("list tags failed: %w", err)
	}
	var matchingTags []string
	if pattern == "" {
		return matchingTags, nil
	}
	for _, tag := range allTags {
		matched, err := path.Match(pattern, tag)
		if err != nil {
			return nil, fmt.Errorf("matching tag %q: invalid glob pattern %q", tag, pattern)
		}
		if matched {
			matchingTags = append(matchingTags, tag)
		}
	}
	return matchingTags, nil
}
