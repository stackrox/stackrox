// Package reposcan provides types and interfaces for scanning container image repositories.
package reposcan

import (
	"context"
	"iter"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/baseimage/tagfetcher"
)

// ScanRequest contains the parameters for scanning a repository.
type ScanRequest struct {
	// Pattern is the tag filtering pattern.
	Pattern string
	// CheckTags is a map of tags to check metadata for updates.
	CheckTags map[string]*storage.BaseImageTag
	// SkipTags is a set of tags to skip.
	SkipTags map[string]struct{}
}

// TagEventType identifies the type of tag change event.
type TagEventType int

const (
	// TagEventMetadata indicates new or changed tag with metadata.
	TagEventMetadata TagEventType = iota
	// TagEventDeleted indicates tag was in cache but not in registry.
	TagEventDeleted
	// TagEventError indicates failed to fetch metadata for this tag.
	TagEventError
)

// TagEvent represents a tag change detected during a scan.
type TagEvent struct {
	// Tag is the image tag name.
	Tag string
	// Type identifies the event type.
	Type TagEventType
	// Metadata contains the tag's manifest metadata.
	Metadata *tagfetcher.TagMetadata
	// Error contains the error (set when Type == TagEventError).
	Error error
}

// Scanner scans a repository for tags.
type Scanner interface {
	// Name returns a short identifier for logging, tracing or metrics (e.g., "local", "delegated").
	Name() string

	// ScanRepository lists and filters tags, yielding events for matching tags.
	// On fatal error (can't list tags, invalid repo), yields (TagEvent{}, err).
	ScanRepository(ctx context.Context, repo *storage.BaseImageRepository, req ScanRequest) iter.Seq2[TagEvent, error]
}
