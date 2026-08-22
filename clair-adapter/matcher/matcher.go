package matcher

import (
	"context"
	"regexp"
	"time"

	"github.com/stackrox/rox/clair-adapter/clairclient"
	"github.com/stackrox/rox/clair-adapter/datastore"
	"github.com/stackrox/rox/clair-adapter/enricher"
	"github.com/stackrox/rox/clair-adapter/mappers"
	"github.com/stackrox/rox/clair-adapter/vulnimporter"
)

var cvePattern = regexp.MustCompile(`CVE-\d{4}-\d+`)

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
	clair             *clairclient.Client
	pipeline          *enricher.Pipeline
	enrichmentFetcher *vulnimporter.EnrichmentFetcher // may be nil
	metadataStore     datastore.MatcherMetadataStore  // may be nil
}

// NewLocalMatcher creates a new matcher that delegates to a Clair HTTP client
// and enriches results using the provided enrichment pipeline.
// The metadataStore parameter is optional (may be nil) and is used to track vulnerability updates.
func NewLocalMatcher(clair *clairclient.Client, pipeline *enricher.Pipeline, metadataStore datastore.MatcherMetadataStore, opts ...LocalMatcherOption) Matcher {
	m := &localMatcher{
		clair:         clair,
		pipeline:      pipeline,
		metadataStore: metadataStore,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// LocalMatcherOption configures a localMatcher.
type LocalMatcherOption func(*localMatcher)

// WithEnrichmentFetcher configures direct DB enrichment fetching.
func WithEnrichmentFetcher(f *vulnimporter.EnrichmentFetcher) LocalMatcherOption {
	return func(m *localMatcher) { m.enrichmentFetcher = f }
}

// GetVulnerabilities retrieves and enriches vulnerability data for a container image.
func (l *localMatcher) GetVulnerabilities(ctx context.Context, hashID string) (*clairclient.VulnerabilityReport, *enricher.EnrichmentResult, error) {
	report, err := l.clair.GetVulnerabilityReport(ctx, clairclient.DigestFromHashID(hashID))
	if err != nil {
		return nil, nil, err
	}

	enrichmentResult, err := l.pipeline.Enrich(ctx, report)
	if err != nil {
		return nil, nil, err
	}

	// Clair doesn't return enrichment data (NVD/EPSS) because it doesn't have
	// the StackRox enricher plugins. Fetch directly from the DB if available.
	if l.enrichmentFetcher != nil {
		cves := extractCVEs(report)
		if len(cves) > 0 {
			nvd := l.enrichmentFetcher.FetchNVD(ctx, cves)
			if len(nvd) > 0 {
				enrichmentResult.NVDVulns = map[string]map[string]*mappers.NVDItem{"db": nvd}
			}
			epss := l.enrichmentFetcher.FetchEPSS(ctx, cves)
			if len(epss) > 0 {
				enrichmentResult.EPSSItems = map[string]map[string]*mappers.EPSSItem{"db": epss}
			}
		}
	}

	return report, enrichmentResult, nil
}

func extractCVEs(report *clairclient.VulnerabilityReport) []string {
	seen := make(map[string]struct{})
	for _, v := range report.Vulnerabilities {
		for _, m := range cvePattern.FindAllString(v.Name, -1) {
			seen[m] = struct{}{}
		}
	}
	cves := make([]string, 0, len(seen))
	for cve := range seen {
		cves = append(cves, cve)
	}
	return cves
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
