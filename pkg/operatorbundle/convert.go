package operatorbundle

import (
	"github.com/stackrox/rox/generated/storage"
)

// ImageCVEsFromStorage extracts the CVE set from a scanned storage.Image. It returns nil
// when the image is nil. Images without scan data yield an ImageCVEs with no CVEs.
func ImageCVEsFromStorage(image *storage.Image) *ImageCVEs {
	if image == nil {
		return nil
	}

	result := &ImageCVEs{
		Repository: repositoryFromImage(image),
		Reference:  image.GetName().GetFullName(),
		Digest:     image.GetId(),
	}

	// De-duplicate CVEs that appear across multiple components, keeping the fixable variant
	// (a non-empty FixedBy) when the same CVE is both fixable and not across components.
	byID := make(map[string]CVE)
	for _, component := range image.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			id := vuln.GetCve()
			if id == "" {
				continue
			}
			cve := CVE{
				ID:       id,
				Severity: vuln.GetSeverity(),
				CVSS:     vuln.GetCvss(),
				FixedBy:  vuln.GetFixedBy(),
			}
			if existing, ok := byID[id]; !ok || (existing.FixedBy == "" && cve.FixedBy != "") {
				byID[id] = cve
			}
		}
	}

	result.CVEs = make([]CVE, 0, len(byID))
	for _, cve := range byID {
		result.CVEs = append(result.CVEs, cve)
	}
	return result
}

// repositoryFromImage derives the tag/digest-stripped repository for pairing. It prefers the
// structured registry+remote and falls back to the full name.
func repositoryFromImage(image *storage.Image) string {
	name := image.GetName()
	if name.GetRegistry() != "" && name.GetRemote() != "" {
		return name.GetRegistry() + "/" + name.GetRemote()
	}
	return RepositoryFromReference(name.GetFullName())
}
