package reposcan

import (
	"context"
	"fmt"
	"path"

	"github.com/stackrox/rox/pkg/registries/types"
)

// ListAndFilterTags lists all tags for the given repository and filters them using
// the [path.Match] glob pattern. It returns the list of matching tags or a nil
// slice if no tags match the pattern. Empty pattern matches no tag.
func ListAndFilterTags(ctx context.Context, registry types.Registry, repo string, pattern string) ([]string, error) {
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
