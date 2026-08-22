package enricher

import (
	"context"
	"log/slog"

	"github.com/stackrox/rox/clair-adapter/clairclient"
	"github.com/stackrox/rox/clair-adapter/enricher/csaf"
	"github.com/stackrox/rox/clair-adapter/enricher/fixedby"
	"github.com/stackrox/rox/clair-adapter/mappers"
)

// EnrichmentResult contains all enrichment data extracted from a vulnerability report.
type EnrichmentResult struct {
	NVDVulns       map[string]map[string]*mappers.NVDItem
	EPSSItems      map[string]map[string]*mappers.EPSSItem
	CSAFAdvisories map[string]*csaf.Advisory
	PkgFixedBy     map[string]string
}

// Pipeline orchestrates the enrichment process for vulnerability reports.
type Pipeline struct {
	csafEnricher *csaf.Enricher
}

// PipelineOption configures a Pipeline.
type PipelineOption func(*Pipeline)

// NewPipeline creates a new enrichment pipeline.
func NewPipeline(opts ...PipelineOption) *Pipeline {
	p := &Pipeline{}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// WithCSAFEnricher configures the pipeline with a CSAF enricher.
func WithCSAFEnricher(e *csaf.Enricher) PipelineOption {
	return func(p *Pipeline) {
		p.csafEnricher = e
	}
}

// Enrich processes a vulnerability report through all enrichment stages.
// Returns combined enrichment data from all sources.
func (p *Pipeline) Enrich(ctx context.Context, vr *clairclient.VulnerabilityReport) (*EnrichmentResult, error) {
	result := &EnrichmentResult{}

	// Extract NVD vulnerability data
	nvdVulns, err := mappers.ExtractNVDVulnerabilities(vr.Enrichments)
	if err != nil {
		return nil, err
	}
	result.NVDVulns = nvdVulns

	// Extract EPSS data
	epssItems, err := mappers.ExtractEPSS(vr.Enrichments)
	if err != nil {
		return nil, err
	}
	result.EPSSItems = epssItems

	// Compute package fixed-by versions
	pkgFixedBy, err := fixedby.Enrich(vr)
	if err != nil {
		return nil, err
	}
	result.PkgFixedBy = pkgFixedBy

	// Apply CSAF enrichment if configured (best-effort)
	if p.csafEnricher != nil {
		csafAdvisories, err := p.csafEnricher.Enrich(vr)
		if err != nil {
			slog.WarnContext(ctx, "CSAF enrichment failed, continuing without it", "error", err)
		} else {
			result.CSAFAdvisories = csafAdvisories
		}
	}

	return result, nil
}
