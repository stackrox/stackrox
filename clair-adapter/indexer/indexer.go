package indexer

import (
	"context"
	"errors"
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

// localIndexer implements the Indexer interface using a Clair HTTP client.
type localIndexer struct {
	clair         *clairclient.Client
	metadataStore datastore.IndexerMetadataStore // may be nil
}

// NewLocalIndexer creates a new indexer that delegates to a Clair HTTP client.
// The metadataStore parameter is optional (may be nil) and is used to track manifest lifecycle.
func NewLocalIndexer(clair *clairclient.Client, metadataStore datastore.IndexerMetadataStore) Indexer {
	return &localIndexer{
		clair:         clair,
		metadataStore: metadataStore,
	}
}

// IndexContainerImage indexes a container image by submitting a manifest to Clair.
// Currently, layers are left empty as full registry interaction is planned for the future.
func (l *localIndexer) IndexContainerImage(ctx context.Context, hashID, imageURL string, opts ...Option) (*clairclient.IndexReport, error) {
	// Apply options (currently not used in manifest construction, reserved for future registry integration)
	var options indexOpts
	for _, opt := range opts {
		opt(&options)
	}

	// Build manifest with empty layers (full registry interaction is a future plan)
	manifest := clairclient.Manifest{
		Hash:   hashID,
		Layers: []clairclient.Layer{},
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
