package indexer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/stackrox/rox/clair-adapter/clairclient"
	"github.com/stackrox/rox/clair-adapter/datastore"
)

// Indexer provides container image indexing operations.
type Indexer interface {
	// IndexContainerImage indexes a container image and returns the index report.
	// hashID is the manifest hash (digest) and imageURL is the container image reference.
	// Options can be provided for registry authentication.
	IndexContainerImage(ctx context.Context, hashID, imageURL string, opts ...Option) (*clairclient.IndexReport, error)

	// GetIndexReport retrieves an existing index report by manifest hash.
	// Returns (report, true, nil) if found, (nil, false, nil) if not found, or (nil, false, err) on error.
	GetIndexReport(ctx context.Context, hashID string) (*clairclient.IndexReport, bool, error)

	// HasIndexReport checks if an index report exists for the given manifest hash.
	HasIndexReport(ctx context.Context, hashID string) (bool, error)
}

// Option configures indexing behavior.
type Option func(*indexOpts)

// indexOpts holds indexing options.
type indexOpts struct {
	username              string
	password              string
	insecureSkipTLSVerify bool
}

// WithBasicAuth configures basic authentication credentials for registry access.
func WithBasicAuth(username, password string) Option {
	return func(opts *indexOpts) {
		opts.username = username
		opts.password = password
	}
}

// WithInsecureSkipTLSVerify configures whether to skip TLS verification for registry access.
func WithInsecureSkipTLSVerify(skip bool) Option {
	return func(opts *indexOpts) {
		opts.insecureSkipTLSVerify = skip
	}
}

// layerFetcher is a function that fetches layer descriptors from a registry.
type layerFetcher func(ctx context.Context, imageURL string, opts indexOpts) ([]clairclient.Layer, error)

// localIndexer implements the Indexer interface using a Clair HTTP client.
type localIndexer struct {
	clair         *clairclient.Client
	metadataStore datastore.IndexerMetadataStore // may be nil
	fetchLayers   layerFetcher                   // defaults to fetchManifestLayers
}

// LocalIndexerOption configures a localIndexer.
type LocalIndexerOption func(*localIndexer)

// WithLayerFetcher overrides the default layer fetcher (used for testing).
func WithLayerFetcher(fetcher layerFetcher) LocalIndexerOption {
	return func(l *localIndexer) {
		l.fetchLayers = fetcher
	}
}

// NewLocalIndexer creates a new indexer that delegates to a Clair HTTP client.
// The metadataStore parameter is optional (may be nil) and is used to track manifest lifecycle.
func NewLocalIndexer(clair *clairclient.Client, metadataStore datastore.IndexerMetadataStore, opts ...LocalIndexerOption) Indexer {
	idx := &localIndexer{
		clair:         clair,
		metadataStore: metadataStore,
		fetchLayers:   fetchManifestLayers,
	}
	for _, opt := range opts {
		opt(idx)
	}
	return idx
}

// IndexContainerImage indexes a container image by submitting a manifest to Clair.
func (l *localIndexer) IndexContainerImage(ctx context.Context, hashID, imageURL string, opts ...Option) (*clairclient.IndexReport, error) {
	// Apply options
	var options indexOpts
	for _, opt := range opts {
		opt(&options)
	}

	// Fetch manifest layers from registry
	layers, err := l.fetchLayers(ctx, imageURL, options)
	if err != nil {
		return nil, fmt.Errorf("fetching manifest layers: %w", err)
	}

	// Build manifest for Clair
	manifest := clairclient.Manifest{
		Hash:   hashID,
		Layers: layers,
	}

	// Submit to Clair for indexing
	report, err := l.clair.CreateIndexReport(ctx, manifest)
	if err != nil {
		return nil, err
	}

	// Store manifest metadata if metadata store is configured
	if l.metadataStore != nil {
		// Set expiration to 24 hours from now
		expiration := time.Now().Add(24 * time.Hour)
		if err := l.metadataStore.StoreManifest(ctx, hashID, expiration); err != nil {
			// Log warning but don't fail the operation
			log.Printf("warning: failed to store manifest metadata for %s: %v", hashID, err)
		}
	}

	return report, nil
}

// GetIndexReport retrieves an index report from Clair by manifest hash.
// Returns (report, true, nil) if found, (nil, false, nil) if not found.
func (l *localIndexer) GetIndexReport(ctx context.Context, hashID string) (*clairclient.IndexReport, bool, error) {
	report, err := l.clair.GetIndexReport(ctx, hashID)
	if err != nil {
		if errors.Is(err, clairclient.ErrNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}

	return report, true, nil
}

// HasIndexReport checks if an index report exists in Clair.
func (l *localIndexer) HasIndexReport(ctx context.Context, hashID string) (bool, error) {
	_, err := l.clair.GetIndexReport(ctx, hashID)
	if err != nil {
		if errors.Is(err, clairclient.ErrNotFound) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
