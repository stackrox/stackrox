package converter

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

// ImageCVEConverter converts NormalizedCVE + Edge data to ImageCVEV2 format.
// Used to maintain API compatibility while using normalized storage internally.
type ImageCVEConverter interface {
	// ToImageCVEV2 converts a NormalizedCVE and its component edge to ImageCVEV2.
	// Returns nil if either input is nil.
	ToImageCVEV2(cve *storage.NormalizedCVE, edge *storage.NormalizedComponentCVEEdge) *storage.ImageCVEV2

	// ToImageCVEV2Batch converts multiple CVE-Edge pairs to ImageCVEV2 objects.
	// Skips any pairs with nil CVE or Edge.
	// The pairs parameter should contain CVE and Edge from the datastore.CVEEdgePair type.
	ToImageCVEV2Batch(cves []*storage.NormalizedCVE, edges []*storage.NormalizedComponentCVEEdge) []*storage.ImageCVEV2
}

// imageConverterImpl implements ImageCVEConverter.
type imageConverterImpl struct{}

// NewImageCVEConverter creates a new ImageCVEConverter instance.
func NewImageCVEConverter() ImageCVEConverter {
	return &imageConverterImpl{}
}

// ToImageCVEV2 converts a NormalizedCVE and its component edge to ImageCVEV2.
func (c *imageConverterImpl) ToImageCVEV2(cve *storage.NormalizedCVE, edge *storage.NormalizedComponentCVEEdge) *storage.ImageCVEV2 {
	if cve == nil || edge == nil {
		return nil
	}

	// Generate composite ID from CVE UUID + component ID.
	id := pgSearch.IDFromPks([]string{cve.GetId(), edge.GetComponentId()})

	// Build CVEInfo from NormalizedCVE fields.
	cveInfo := &storage.CVEInfo{
		Cve:         cve.GetCveName(),
		Summary:     cve.GetSummary(),
		Link:        cve.GetLink(),
		PublishedOn: cve.GetPublishedOn(),
		CreatedAt:   cve.GetCreatedAt(),
	}

	// Build Advisory if name or link present.
	var advisory *storage.Advisory
	if cve.GetAdvisoryName() != "" || cve.GetAdvisoryLink() != "" {
		advisory = &storage.Advisory{
			Name: cve.GetAdvisoryName(),
			Link: cve.GetAdvisoryLink(),
		}
	}

	// Build ImageCVEV2.
	result := &storage.ImageCVEV2{
		Id:                    id,
		CveBaseInfo:           cveInfo,
		Cvss:                  cve.GetCvssV3(),
		Severity:              parseSeverity(cve.GetSeverity()),
		Nvdcvss:               cve.GetNvdCvssV3(),
		FirstImageOccurrence:  edge.GetFirstSystemOccurrence(),
		State:                 parseVulnerabilityState(edge.GetState()),
		IsFixable:             edge.GetIsFixable(),
		ComponentId:           edge.GetComponentId(),
		Advisory:              advisory,
		FixAvailableTimestamp: edge.GetFixAvailableAt(),
		Datasource:            cve.GetSource(),
	}

	// Set fixed_by if available.
	if edge.GetFixedBy() != "" {
		result.HasFixedBy = &storage.ImageCVEV2_FixedBy{
			FixedBy: edge.GetFixedBy(),
		}
	}

	return result
}

// ToImageCVEV2Batch converts multiple CVE-Edge pairs to ImageCVEV2 objects.
// CVEs and edges should be in matching order.
func (c *imageConverterImpl) ToImageCVEV2Batch(cves []*storage.NormalizedCVE, edges []*storage.NormalizedComponentCVEEdge) []*storage.ImageCVEV2 {
	if len(cves) == 0 || len(edges) == 0 {
		return nil
	}

	// Use the minimum length to avoid index out of bounds.
	minLen := len(cves)
	if len(edges) < minLen {
		minLen = len(edges)
	}

	results := make([]*storage.ImageCVEV2, 0, minLen)
	for i := 0; i < minLen; i++ {
		if converted := c.ToImageCVEV2(cves[i], edges[i]); converted != nil {
			results = append(results, converted)
		}
	}
	return results
}

// parseSeverity converts severity string to VulnerabilitySeverity enum.
// Handles both uppercase and lowercase inputs.
// Returns UNKNOWN_VULNERABILITY_SEVERITY for invalid or empty values.
func parseSeverity(s string) storage.VulnerabilitySeverity {
	switch strings.ToUpper(s) {
	case "CRITICAL":
		return storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
	case "HIGH", "IMPORTANT":
		return storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY
	case "MEDIUM", "MODERATE":
		return storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY
	case "LOW":
		return storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY
	default:
		return storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	}
}

// parseVulnerabilityState converts state string to VulnerabilityState enum.
// Handles both uppercase and lowercase inputs.
// Returns OBSERVED for invalid or empty values (default state).
func parseVulnerabilityState(s string) storage.VulnerabilityState {
	switch strings.ToUpper(s) {
	case "DEFERRED":
		return storage.VulnerabilityState_DEFERRED
	case "FALSE_POSITIVE":
		return storage.VulnerabilityState_FALSE_POSITIVE
	case "OBSERVED":
		return storage.VulnerabilityState_OBSERVED
	default:
		// Default to OBSERVED (as per proto comment: "By default, vulnerabilities are observed").
		return storage.VulnerabilityState_OBSERVED
	}
}
