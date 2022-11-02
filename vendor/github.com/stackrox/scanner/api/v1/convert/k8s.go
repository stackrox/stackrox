package convert

import (
	log "github.com/sirupsen/logrus"
	"github.com/stackrox/k8s-cves/pkg/validation"
	"github.com/stackrox/rox/pkg/stringutils"
	v1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stackrox/scanner/pkg/types"
)

// K8sVulnerabilities converts k8s cve schema to vulnerability.
func K8sVulnerabilities(version string, k8sVulns []*validation.CVESchema) ([]*v1.Vulnerability, error) {
	vulns := make([]*v1.Vulnerability, 0, len(k8sVulns))
	for _, v := range k8sVulns {
		m, err := types.ConvertMetadataFromK8s(v)
		if err != nil {
			log.Errorf("unable to convert metadata for %s: %v", v.CVE, err)
			continue
		}
		if m.IsNilOrEmpty() {
			log.Warnf("nil or empty metadata for %s", v.CVE)
			continue
		}

		link := stringutils.OrDefault(v.IssueURL, v.URL)
		fixedBy, err := GetFixedBy(version, v)
		if err != nil {
			log.Errorf("unable to get fixedBy for %s: %v", v.CVE, err)
			continue
		}
		vulns = append(vulns, &v1.Vulnerability{
			Name:        v.CVE,
			Description: v.Description,
			Link:        link,
			MetadataV2:  Metadata(m),
			FixedBy:     fixedBy,
			Severity:    string(DatabaseSeverityToSeverity(m.GetDatabaseSeverity())),
		})
	}
	return vulns, nil
}
