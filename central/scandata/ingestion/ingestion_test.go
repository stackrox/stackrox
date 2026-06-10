package ingestion

import (
	"testing"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertVulnerabilities_RHSAAdvisory(t *testing.T) {
	// Simulate a Red Hat VEX vuln with an RHSA advisory populated by Scanner V4
	allVulns := map[string]*v4.VulnerabilityReport_Vulnerability{
		"rhel-vex-1": {
			Name:    "CVE-2026-0968",
			CveName: "CVE-2026-0968",
			Advisory: &v4.VulnerabilityReport_Advisory{
				Name: "RHSA-2026:18160",
				Link: "https://access.redhat.com/errata/RHSA-2026:18160",
			},
			AdvisoryId:         "CVE-2026-0968",
			SourceName:         "Red Hat VEX",
			NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_IMPORTANT,
			Description:        "libssh vulnerability",
			Link:               "https://access.redhat.com/security/cve/cve-2026-0968 https://access.redhat.com/errata/RHSA-2026:18160",
			FixedInVersion:     "0.10.6-2.el9",
			Updater:            "RHEL9-rhel-9-including-unpatched-rhel-9-cpe:/a:redhat:enterprise_linux:9::appstream-2026-06-03T20:09:16Z",
		},
	}
	vulnIDs := []string{"rhel-vex-1"}

	findings := convertVulnerabilities("scan-1", "image-1", "comp-1", allVulns, vulnIDs, "rhel:9")

	require.Len(t, findings, 2, "Expected VEX finding + RHSA finding")

	// First finding: Red Hat VEX
	vex := findings[0]
	assert.Equal(t, "CVE-2026-0968", vex.CveName)
	assert.Equal(t, "CVE-2026-0968", vex.AdvisoryId)
	assert.Equal(t, "Red Hat VEX", vex.SourceName)

	// Second finding: RHSA
	rhsa := findings[1]
	assert.Equal(t, "CVE-2026-0968", rhsa.CveName, "RHSA finding should reference same CVE")
	assert.Equal(t, "RHSA-2026:18160", rhsa.AdvisoryId, "RHSA finding should have RHSA as advisory ID")
	assert.Equal(t, "Red Hat Advisory", rhsa.SourceName)
	assert.Contains(t, rhsa.Links, "https://access.redhat.com/errata/RHSA-2026:18160")
}

func TestConvertVulnerabilities_NoAdvisory(t *testing.T) {
	// A Go vuln with no Advisory field should produce exactly one finding
	allVulns := map[string]*v4.VulnerabilityReport_Vulnerability{
		"osv-go-1": {
			Name:               "GO-2024-1234",
			CveName:            "CVE-2024-1234",
			AdvisoryId:         "GO-2024-1234",
			SourceName:         "Go Vulnerability DB",
			NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_IMPORTANT,
			Updater:            "osv/Go",
		},
	}
	vulnIDs := []string{"osv-go-1"}

	findings := convertVulnerabilities("scan-1", "image-1", "comp-1", allVulns, vulnIDs, "")

	require.Len(t, findings, 1, "Non-Red Hat vuln should produce exactly one finding")
	assert.Equal(t, "GO-2024-1234", findings[0].AdvisoryId)
}

func TestConvertVulnerabilities_VEXWithoutRHSA(t *testing.T) {
	// A Red Hat VEX vuln WITHOUT Advisory populated (no CSAF data) should produce one finding
	allVulns := map[string]*v4.VulnerabilityReport_Vulnerability{
		"rhel-vex-2": {
			Name:               "CVE-2026-9999",
			CveName:            "CVE-2026-9999",
			AdvisoryId:         "CVE-2026-9999",
			SourceName:         "Red Hat VEX",
			NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_MODERATE,
			Updater:            "RHEL9-rhel-9-including-unpatched",
		},
	}
	vulnIDs := []string{"rhel-vex-2"}

	findings := convertVulnerabilities("scan-1", "image-1", "comp-1", allVulns, vulnIDs, "rhel:9")

	require.Len(t, findings, 1, "VEX vuln without Advisory should produce one finding")
	assert.Equal(t, "CVE-2026-9999", findings[0].AdvisoryId)
	assert.Equal(t, "Red Hat VEX", findings[0].SourceName)
}
