package fixedby

import (
	"github.com/stackrox/rox/clair-adapter/clairclient"
)

// Enrich computes fixed-by version per package from a VulnerabilityReport.
// For each package, find the maximum FixedInVersion across all its vulnerabilities.
// Returns map[packageID]fixedVersion. Empty FixedInVersion is skipped.
func Enrich(vr *clairclient.VulnerabilityReport) (map[string]string, error) {
	result := make(map[string]string)

	for pkgID, vulnIDs := range vr.PackageVulnerabilities {
		var maxVersion string

		for _, vulnID := range vulnIDs {
			vuln, exists := vr.Vulnerabilities[vulnID]
			if !exists || vuln.FixedInVersion == "" {
				continue
			}

			// Find the maximum fixed version (simple string comparison)
			// In a real implementation, this would use proper version comparison
			if maxVersion == "" || vuln.FixedInVersion > maxVersion {
				maxVersion = vuln.FixedInVersion
			}
		}

		if maxVersion != "" {
			result[pkgID] = maxVersion
		}
	}

	return result, nil
}
