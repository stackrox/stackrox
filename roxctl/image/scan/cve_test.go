package scan

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestNewCVESummaryForPrinting(t *testing.T) {
	vulnsWithAllSeverities := []*storage.EmbeddedVulnerability{
		{
			Cve:        "CVE-TEST-2",
			Summary:    "CVE Test 2",
			Link:       "cve-link-2",
			Severity:   storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.0"},
		},
		{
			Cve:        "CVE-TEST-1",
			Summary:    "CVE Test 1",
			Link:       "cve-link-1",
			Severity:   storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.2"},
		},
		{
			Cve:        "CVE-TEST-3",
			Summary:    "CVE Test 3",
			Link:       "cve-link-3",
			Severity:   storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.3"},
		},
		{
			Cve:        "CVE-TEST-4",
			Summary:    "CVE Test 4",
			Link:       "cve-link-4",
			Severity:   storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.4"},
		},
	}

	// Expected vulns and components without filtering.
	expectedVulnsComponentA := []cveVulnerabilityJSON{
		{CveID: "CVE-TEST-1", CveSeverity: criticalCVESeverity, CveInfo: "cve-link-1", ComponentName: "componentA", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "1.2"},
		{CveID: "CVE-TEST-3", CveSeverity: importantCVESeverity, CveInfo: "cve-link-3", ComponentName: "componentA", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "1.3"},
		{CveID: "CVE-TEST-4", CveSeverity: moderateCVESeverity, CveInfo: "cve-link-4", ComponentName: "componentA", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "1.4"},
		{CveID: "CVE-TEST-2", CveSeverity: lowCVESeverity, CveInfo: "cve-link-2", ComponentName: "componentA", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "1.0"},
	}
	expectedVulnsComponentB := []cveVulnerabilityJSON{
		{CveID: "CVE-TEST-1", CveSeverity: criticalCVESeverity, CveInfo: "cve-link-1", ComponentName: "componentB", ComponentVersion: "1.0.0-2", ComponentFixedVersion: "1.2"},
		{CveID: "CVE-TEST-3", CveSeverity: importantCVESeverity, CveInfo: "cve-link-3", ComponentName: "componentB", ComponentVersion: "1.0.0-2", ComponentFixedVersion: "1.3"},
		{CveID: "CVE-TEST-4", CveSeverity: moderateCVESeverity, CveInfo: "cve-link-4", ComponentName: "componentB", ComponentVersion: "1.0.0-2", ComponentFixedVersion: "1.4"},
		{CveID: "CVE-TEST-2", CveSeverity: lowCVESeverity, CveInfo: "cve-link-2", ComponentName: "componentB", ComponentVersion: "1.0.0-2", ComponentFixedVersion: "1.0"},
	}
	expectedVulnsComponentC := []cveVulnerabilityJSON{
		{CveID: "CVE-TEST-1", CveSeverity: criticalCVESeverity, CveInfo: "cve-link-1", ComponentName: "componentC", ComponentVersion: "1.0.0-3", ComponentFixedVersion: "1.2"},
		{CveID: "CVE-TEST-3", CveSeverity: importantCVESeverity, CveInfo: "cve-link-3", ComponentName: "componentC", ComponentVersion: "1.0.0-3", ComponentFixedVersion: "1.3"},
		{CveID: "CVE-TEST-4", CveSeverity: moderateCVESeverity, CveInfo: "cve-link-4", ComponentName: "componentC", ComponentVersion: "1.0.0-3", ComponentFixedVersion: "1.4"},
		{CveID: "CVE-TEST-2", CveSeverity: lowCVESeverity, CveInfo: "cve-link-2", ComponentName: "componentC", ComponentVersion: "1.0.0-3", ComponentFixedVersion: "1.0"},
	}
	expectedVulnsComponentD := []cveVulnerabilityJSON{
		{
			CveID: "CVE-TEST-10", CveSeverity: criticalCVESeverity, CveInfo: "cve-link-10", ComponentName: "componentD", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "3.0",
		},
	}

	// Expected vulns and components when filtered.
	expectedVulnsComponentAFiltered := []cveVulnerabilityJSON{
		{CveID: "CVE-TEST-3", CveSeverity: importantCVESeverity, CveInfo: "cve-link-3", ComponentName: "componentA", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "1.3"},
		{CveID: "CVE-TEST-4", CveSeverity: moderateCVESeverity, CveInfo: "cve-link-4", ComponentName: "componentA", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "1.4"},
	}
	expectedVulnsComponentBFiltered := []cveVulnerabilityJSON{
		{CveID: "CVE-TEST-3", CveSeverity: importantCVESeverity, CveInfo: "cve-link-3", ComponentName: "componentB", ComponentVersion: "1.0.0-2", ComponentFixedVersion: "1.3"},
		{CveID: "CVE-TEST-4", CveSeverity: moderateCVESeverity, CveInfo: "cve-link-4", ComponentName: "componentB", ComponentVersion: "1.0.0-2", ComponentFixedVersion: "1.4"},
	}
	expectedVulnsComponentCFiltered := []cveVulnerabilityJSON{
		{CveID: "CVE-TEST-3", CveSeverity: importantCVESeverity, CveInfo: "cve-link-3", ComponentName: "componentC", ComponentVersion: "1.0.0-3", ComponentFixedVersion: "1.3"},
		{CveID: "CVE-TEST-4", CveSeverity: moderateCVESeverity, CveInfo: "cve-link-4", ComponentName: "componentC", ComponentVersion: "1.0.0-3", ComponentFixedVersion: "1.4"},
	}

	cases := map[string]struct {
		scan           *storage.ImageScan
		severities     []string
		expectedOutput *cveJSONResult
	}{
		"empty img scan results": {
			scan: &storage.ImageScan{
				Components: nil,
			},
			severities: []string{lowCVESeverity.String(), moderateCVESeverity.String(), importantCVESeverity.String(),
				criticalCVESeverity.String()},
			expectedOutput: &cveJSONResult{
				Result: cveJSONStructure{
					Summary: map[string]int{
						"TOTAL-VULNERABILITIES": 0,
						"TOTAL-COMPONENTS":      0,
						"LOW":                   0,
						"MODERATE":              0,
						"IMPORTANT":             0,
						"CRITICAL":              0,
					},
					Vulnerabilities: nil,
				},
			},
		},
		"duplicated CVEs across multiple components": {
			severities: []string{lowCVESeverity.String(), moderateCVESeverity.String(), importantCVESeverity.String(),
				criticalCVESeverity.String()},
			scan: &storage.ImageScan{
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Name:    "dbus",
						Version: "1:1.12.20-6.el9.x86_64",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve:      "CVE-2022-42010",
								Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
							},
						},
					},
					{
						Name:    "dbus-common",
						Version: "1:1.12.20-6.el9.noarch",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve:      "CVE-2022-42010",
								Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
							},
						},
					},
					{
						Name:    "dbus-libs",
						Version: "1:1.12.20-6.el9.x86_64",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve:      "CVE-2022-42010",
								Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
							},
						},
					},
				},
			},
			expectedOutput: &cveJSONResult{
				Result: cveJSONStructure{
					Summary: map[string]int{
						"TOTAL-VULNERABILITIES": 1,
						"TOTAL-COMPONENTS":      3,
						"LOW":                   0,
						"MODERATE":              1,
						"IMPORTANT":             0,
						"CRITICAL":              0,
					},
					Vulnerabilities: []cveVulnerabilityJSON{
						{
							CveID:            "CVE-2022-42010",
							CveSeverity:      moderateCVESeverity,
							ComponentName:    "dbus",
							ComponentVersion: "1:1.12.20-6.el9.x86_64",
						},
						{
							CveID:            "CVE-2022-42010",
							CveSeverity:      moderateCVESeverity,
							ComponentName:    "dbus-common",
							ComponentVersion: "1:1.12.20-6.el9.noarch",
						},
						{
							CveID:            "CVE-2022-42010",
							CveSeverity:      moderateCVESeverity,
							ComponentName:    "dbus-libs",
							ComponentVersion: "1:1.12.20-6.el9.x86_64",
						},
					},
				},
			},
		},
		"components with vulnerabilities of all severity": {
			severities: []string{lowCVESeverity.String(), moderateCVESeverity.String(), importantCVESeverity.String(),
				criticalCVESeverity.String()},
			scan: &storage.ImageScan{
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Name:    "componentD",
						Version: "1.0.0-1",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve:        "CVE-TEST-10",
								Summary:    "CVE Test 10",
								Link:       "cve-link-10",
								Severity:   storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
								SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "3.0"},
							},
						},
						FixedBy: "3.0",
					},
					{
						Name:    "componentA",
						Version: "1.0.0-1",
						Vulns:   vulnsWithAllSeverities,
						FixedBy: "2.0",
					},
					{
						Name:    "componentC",
						Version: "1.0.0-3",
						FixedBy: "2.0",
						Vulns:   vulnsWithAllSeverities,
					},
					{
						Name:    "componentB",
						Version: "1.0.0-2",
						Vulns:   vulnsWithAllSeverities,
						FixedBy: "2.0",
					},
				},
			},
			expectedOutput: &cveJSONResult{
				Result: cveJSONStructure{
					Summary: map[string]int{
						"TOTAL-VULNERABILITIES": 5,
						"TOTAL-COMPONENTS":      4,
						"LOW":                   1,
						"MODERATE":              1,
						"IMPORTANT":             1,
						"CRITICAL":              2,
					},
					Vulnerabilities: append(expectedVulnsComponentA, append(expectedVulnsComponentB, append(expectedVulnsComponentC, expectedVulnsComponentD...)...)...),
				},
			},
		},
		"components with vulnerabilities of all severity but filtering": {
			severities: []string{moderateCVESeverity.String(), importantCVESeverity.String()},
			scan: &storage.ImageScan{
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Name:    "componentD",
						Version: "1.0.0-1",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve:        "CVE-TEST-10",
								Summary:    "CVE Test 10",
								Link:       "cve-link-10",
								Severity:   storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
								SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "3.0"},
							},
						},
						FixedBy: "3.0",
					},
					{
						Name:    "componentA",
						Version: "1.0.0-1",
						Vulns:   vulnsWithAllSeverities,
						FixedBy: "2.0",
					},
					{
						Name:    "componentC",
						Version: "1.0.0-3",
						FixedBy: "2.0",
						Vulns:   vulnsWithAllSeverities,
					},
					{
						Name:    "componentB",
						Version: "1.0.0-2",
						Vulns:   vulnsWithAllSeverities,
						FixedBy: "2.0",
					},
				},
			},
			expectedOutput: &cveJSONResult{
				Result: cveJSONStructure{
					Summary: map[string]int{
						"TOTAL-VULNERABILITIES": 2,
						"TOTAL-COMPONENTS":      3,
						"LOW":                   0,
						"MODERATE":              1,
						"IMPORTANT":             1,
						"CRITICAL":              0,
					},
					Vulnerabilities: append(expectedVulnsComponentAFiltered, append(expectedVulnsComponentBFiltered, expectedVulnsComponentCFiltered...)...),
				},
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			cveSummary := newCVESummaryForPrinting(c.scan, c.severities)
			assert.Equal(t, c.expectedOutput, cveSummary)
		})
	}
}
