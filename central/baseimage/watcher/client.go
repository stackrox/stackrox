package watcher

import (
	"context"
	"iter"

	"github.com/stackrox/rox/generated/storage"
)

// ScanRequest contains the pattern for scanning.
type ScanRequest struct {
	// Pattern is the tag filtering pattern.
	Pattern string
	// CheckTags is a set of tags to check metadata for updates.
	CheckTags map[string]struct{}
	// SkipTags is a set of tags to skip.
	SkipTags map[string]struct{}
}

// TagEvent represents a discovered tag during a scan.
type TagEvent struct {
	Tag string
}

// RepositoryClient scans a repository for tags.
type RepositoryClient interface {
	// Name returns a short identifier for logging, tracing or metrics (e.g., "local", "delegated").
	Name() string

	// ScanRepository lists and filters tags, yielding events for matching tags.
	// On fatal error (can't list tags, invalid repo), yields (TagEvent{}, err).
	ScanRepository(ctx context.Context, repo *storage.BaseImageRepository, req ScanRequest) iter.Seq2[TagEvent, error]
}
