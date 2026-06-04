package matcher

import (
	"context"
	"strings"
	"time"

	"github.com/stackrox/rox/clair-adapter/clairclient"
	"github.com/stackrox/rox/clair-adapter/datastore"
	"github.com/stackrox/rox/clair-adapter/enricher"
)

func clairDigest(hashID string) string {
	if i := strings.LastIndex(hashID, "sha256:"); i > 0 {
		return hashID[i:]
	}
	return hashID
}

// Matcher provides vulnerability matching operations.
type Matcher interface {
	// GetVulnerabilities retrieves vulnerabilities for a container image by manifest hash.
	// Returns the raw vulnerability report from Clair and the enrichment result.
	GetVulnerabilities(ctx context.Context, hashID string) (*clairclient.VulnerabilityReport, *enricher.EnrichmentResult, error)

	// GetLastVulnerabilityUpdate returns the timestamp of the most recent vulnerability database update.
	GetLastVulnerabilityUpdate(ctx context.Context) (time.Time, error)
}

// localMatcher implements the Matcher interface using a Clair HTTP client.
type localMatcher struct {
	clair         *clairclient.Client
	pipeline      *enricher.Pipeline
	metadataStore datastore.MatcherMetadataStore // may be nil
}

// NewLocalMatcher creates a new matcher that delegates to a Clair HTTP client
// and enriches results using the provided enrichment pipeline.
// The metadataStore parameter is optional (may be nil) and is used to track vulnerability updates.
func NewLocalMatcher(clair *clairclient.Client, pipeline *enricher.Pipeline, metadataStore datastore.MatcherMetadataStore) Matcher {
	return &localMatcher{
		clair:         clair,
		pipeline:      pipeline,
		metadataStore: metadataStore,
	}
}

// GetVulnerabilities retrieves and enriches vulnerability data for a container image.
func (l *localMatcher) GetVulnerabilities(ctx context.Context, hashID string) (*clairclient.VulnerabilityReport, *enricher.EnrichmentResult, error) {
	// Get vulnerability report from Clair
	report, err := l.clair.GetVulnerabilityReport(ctx, clairDigest(hashID))
	if err != nil {
		return nil, nil, err
	}

	// Enrich the report using the pipeline
	enrichmentResult, err := l.pipeline.Enrich(report)
	if err != nil {
		return nil, nil, err
	}

	return report, enrichmentResult, nil
}

// GetLastVulnerabilityUpdate returns the most recent vulnerability database update timestamp.
// If a metadata store is configured, it is used. Otherwise, Clair's update operations are queried.
func (l *localMatcher) GetLastVulnerabilityUpdate(ctx context.Context) (time.Time, error) {
	// Use metadata store if available
	if l.metadataStore != nil {
		return l.metadataStore.GetLastVulnerabilityUpdate(ctx)
	}

	// Otherwise, query Clair for update operations
	operations, err := l.clair.GetUpdateOperations(ctx)
	if err != nil {
		return time.Time{}, err
	}

	// Find the latest date across all operations
	var latest time.Time
	for _, ops := range operations {
		for _, op := range ops {
			if op.Date.After(latest) {
				latest = op.Date
			}
		}
	}

	return latest, nil
}
