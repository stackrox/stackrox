package datastore

import (
	"context"
	"time"
)

// IndexerMetadataStore manages manifest lifecycle tracking in the indexer database.
type IndexerMetadataStore interface {
	// StoreManifest records or updates a manifest with its expiration time.
	StoreManifest(ctx context.Context, manifestID string, expiration time.Time) error

	// ManifestExists checks if a manifest is present in the metadata store.
	ManifestExists(ctx context.Context, manifestID string) (bool, error)

	// GCManifests deletes up to limit expired manifests older than the given expiration time.
	// Returns the list of deleted manifest IDs.
	GCManifests(ctx context.Context, expiration time.Time, limit int) ([]string, error)
}

// MatcherMetadataStore manages vulnerability update tracking in the matcher database.
type MatcherMetadataStore interface {
	// GetLastVulnerabilityUpdate returns the earliest vulnerability update timestamp across all bundles.
	// Returns zero time if no updates have been recorded.
	GetLastVulnerabilityUpdate(ctx context.Context) (time.Time, error)

	// SetLastVulnerabilityUpdate records the update timestamp for a specific vulnerability bundle.
	SetLastVulnerabilityUpdate(ctx context.Context, bundle string, lastUpdate time.Time) error
}
