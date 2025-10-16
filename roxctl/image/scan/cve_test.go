package scan

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestNewCVESummaryForPrinting(t *testing.T) {
	vulnsWithAllSeverities := []*storage.EmbeddedVulnerability{
		storage.EmbeddedVulnerability_builder{
			Cve:      "CVE-TEST-2",
			Summary:  "CVE Test 2",
			Link:     "cve-link-2",
			Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
			Cvss:     1.2,
			FixedBy:  proto.String("1.0"),
		}.Build(),
		storage.EmbeddedVulnerability_builder{
			Cve:      "CVE-TEST-1",
			Summary:  "CVE Test 1",
			Link:     "cve-link-1",
			Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
			Cvss:     10.0,
			FixedBy:  proto.String("1.2"),
		}.Build(),
		storage.EmbeddedVulnerability_builder{
			Cve:      "CVE-TEST-3",
			Summary:  "CVE Test 3",
			Link:     "cve-link-3",
			Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
			Cvss:     7.5,
			FixedBy:  proto.String("1.3"),
		}.Build(),
		storage.EmbeddedVulnerability_builder{
			Cve:      "CVE-TEST-4",
			Summary:  "CVE Test 4",
			Link:     "cve-link-4",
			Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
			FixedBy:  proto.String("1.4"),
		}.Build(),
		storage.EmbeddedVulnerability_builder{
			Cve:      "CVE-TEST-5",
			Summary:  "CVE Test 5",
			Link:     "cve-link-5",
			Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
			Cvss:     8.1,
			FixedBy:  proto.String("1.5"),
			Advisory: storage.Advisory_builder{
				Name: "ADVISORY-TEST-5",
				Link: "advisory-link-5",
			}.Build(),
		}.Build(),
		storage.EmbeddedVulnerability_builder{
			Cve:      "CVE-TEST-6",
			Summary:  "CVE Test 6",
			Link:     "cve-link-6",
			Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
			FixedBy:  proto.String("1.6"),
			Advisory: storage.Advisory_builder{
				Name: "ADVISORY-TEST-6",
				Link: "advisory-link-6",
			}.Build(),
		}.Build(),
	}

	// Expected vulns and components without filtering.
	expectedVulnsComponentA := []cveVulnerabilityJSON{
		{CveID: "CVE-TEST-1", CveSeverity: criticalCVESeverity, CveInfo: "cve-link-1", CveCVSS: 10.0, ComponentName: "componentA", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "1.2"},
		{CveID: "CVE-TEST-5", CveSeverity: importantCVESeverity, CveInfo: "cve-link-5", CveCVSS: 8.1, ComponentName: "componentA", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "1.5", AdvisoryID: "ADVISORY-TEST-5", AdvisoryInfo: "advisory-link-5"},
		{CveID: "CVE-TEST-3", CveSeverity: importantCVESeverity, CveInfo: "cve-link-3", CveCVSS: 7.5, ComponentName: "componentA", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "1.3"},
		{CveID: "CVE-TEST-4", CveSeverity: moderateCVESeverity, CveInfo: "cve-link-4", ComponentName: "componentA", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "1.4"},
		{CveID: "CVE-TEST-2", CveSeverity: lowCVESeverity, CveInfo: "cve-link-2", CveCVSS: 1.2, ComponentName: "componentA", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "1.0"},
		{CveID: "CVE-TEST-6", CveSeverity: lowCVESeverity, CveInfo: "cve-link-6", ComponentName: "componentA", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "1.6", AdvisoryID: "ADVISORY-TEST-6", AdvisoryInfo: "advisory-link-6"},
	}
	expectedVulnsComponentB := []cveVulnerabilityJSON{
		{CveID: "CVE-TEST-1", CveSeverity: criticalCVESeverity, CveInfo: "cve-link-1", CveCVSS: 10.0, ComponentName: "componentB", ComponentVersion: "1.0.0-2", ComponentFixedVersion: "1.2"},
		{CveID: "CVE-TEST-5", CveSeverity: importantCVESeverity, CveInfo: "cve-link-5", CveCVSS: 8.1, ComponentName: "componentB", ComponentVersion: "1.0.0-2", ComponentFixedVersion: "1.5", AdvisoryID: "ADVISORY-TEST-5", AdvisoryInfo: "advisory-link-5"},
		{CveID: "CVE-TEST-3", CveSeverity: importantCVESeverity, CveInfo: "cve-link-3", CveCVSS: 7.5, ComponentName: "componentB", ComponentVersion: "1.0.0-2", ComponentFixedVersion: "1.3"},
		{CveID: "CVE-TEST-4", CveSeverity: moderateCVESeverity, CveInfo: "cve-link-4", ComponentName: "componentB", ComponentVersion: "1.0.0-2", ComponentFixedVersion: "1.4"},
		{CveID: "CVE-TEST-2", CveSeverity: lowCVESeverity, CveInfo: "cve-link-2", CveCVSS: 1.2, ComponentName: "componentB", ComponentVersion: "1.0.0-2", ComponentFixedVersion: "1.0"},
		{CveID: "CVE-TEST-6", CveSeverity: lowCVESeverity, CveInfo: "cve-link-6", ComponentName: "componentB", ComponentVersion: "1.0.0-2", ComponentFixedVersion: "1.6", AdvisoryID: "ADVISORY-TEST-6", AdvisoryInfo: "advisory-link-6"},
	}
	expectedVulnsComponentC := []cveVulnerabilityJSON{
		{CveID: "CVE-TEST-1", CveSeverity: criticalCVESeverity, CveInfo: "cve-link-1", CveCVSS: 10.0, ComponentName: "componentC", ComponentVersion: "1.0.0-3", ComponentFixedVersion: "1.2"},
		{CveID: "CVE-TEST-5", CveSeverity: importantCVESeverity, CveInfo: "cve-link-5", CveCVSS: 8.1, ComponentName: "componentC", ComponentVersion: "1.0.0-3", ComponentFixedVersion: "1.5", AdvisoryID: "ADVISORY-TEST-5", AdvisoryInfo: "advisory-link-5"},
		{CveID: "CVE-TEST-3", CveSeverity: importantCVESeverity, CveInfo: "cve-link-3", CveCVSS: 7.5, ComponentName: "componentC", ComponentVersion: "1.0.0-3", ComponentFixedVersion: "1.3"},
		{CveID: "CVE-TEST-4", CveSeverity: moderateCVESeverity, CveInfo: "cve-link-4", ComponentName: "componentC", ComponentVersion: "1.0.0-3", ComponentFixedVersion: "1.4"},
		{CveID: "CVE-TEST-2", CveSeverity: lowCVESeverity, CveInfo: "cve-link-2", CveCVSS: 1.2, ComponentName: "componentC", ComponentVersion: "1.0.0-3", ComponentFixedVersion: "1.0"},
		{CveID: "CVE-TEST-6", CveSeverity: lowCVESeverity, CveInfo: "cve-link-6", ComponentName: "componentC", ComponentVersion: "1.0.0-3", ComponentFixedVersion: "1.6", AdvisoryID: "ADVISORY-TEST-6", AdvisoryInfo: "advisory-link-6"},
	}
	expectedVulnsComponentD := []cveVulnerabilityJSON{
		{
			CveID: "CVE-TEST-10", CveSeverity: criticalCVESeverity, CveInfo: "cve-link-10", ComponentName: "componentD", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "3.0",
		},
	}

	// Expected vulns and components when filtered.
	expectedVulnsComponentAFiltered := []cveVulnerabilityJSON{
		{CveID: "CVE-TEST-5", CveSeverity: importantCVESeverity, CveInfo: "cve-link-5", CveCVSS: 8.1, ComponentName: "componentA", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "1.5", AdvisoryID: "ADVISORY-TEST-5", AdvisoryInfo: "advisory-link-5"},
		{CveID: "CVE-TEST-3", CveSeverity: importantCVESeverity, CveInfo: "cve-link-3", CveCVSS: 7.5, ComponentName: "componentA", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "1.3"},
		{CveID: "CVE-TEST-4", CveSeverity: moderateCVESeverity, CveInfo: "cve-link-4", ComponentName: "componentA", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "1.4"},
	}
	expectedVulnsComponentBFiltered := []cveVulnerabilityJSON{
		{CveID: "CVE-TEST-5", CveSeverity: importantCVESeverity, CveInfo: "cve-link-5", CveCVSS: 8.1, ComponentName: "componentB", ComponentVersion: "1.0.0-2", ComponentFixedVersion: "1.5", AdvisoryID: "ADVISORY-TEST-5", AdvisoryInfo: "advisory-link-5"},
		{CveID: "CVE-TEST-3", CveSeverity: importantCVESeverity, CveInfo: "cve-link-3", CveCVSS: 7.5, ComponentName: "componentB", ComponentVersion: "1.0.0-2", ComponentFixedVersion: "1.3"},
		{CveID: "CVE-TEST-4", CveSeverity: moderateCVESeverity, CveInfo: "cve-link-4", ComponentName: "componentB", ComponentVersion: "1.0.0-2", ComponentFixedVersion: "1.4"},
	}
	expectedVulnsComponentCFiltered := []cveVulnerabilityJSON{
		{CveID: "CVE-TEST-5", CveSeverity: importantCVESeverity, CveInfo: "cve-link-5", CveCVSS: 8.1, ComponentName: "componentC", ComponentVersion: "1.0.0-3", ComponentFixedVersion: "1.5", AdvisoryID: "ADVISORY-TEST-5", AdvisoryInfo: "advisory-link-5"},
		{CveID: "CVE-TEST-3", CveSeverity: importantCVESeverity, CveInfo: "cve-link-3", CveCVSS: 7.5, ComponentName: "componentC", ComponentVersion: "1.0.0-3", ComponentFixedVersion: "1.3"},
		{CveID: "CVE-TEST-4", CveSeverity: moderateCVESeverity, CveInfo: "cve-link-4", ComponentName: "componentC", ComponentVersion: "1.0.0-3", ComponentFixedVersion: "1.4"},
	}

	cases := map[string]struct {
		scan           *storage.ImageScan
		severities     []string
		expectedOutput *cveJSONResult
	}{
		"empty img scan results": {
			scan: storage.ImageScan_builder{
				Components: nil,
			}.Build(),
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
			scan: storage.ImageScan_builder{
				Components: []*storage.EmbeddedImageScanComponent{
					storage.EmbeddedImageScanComponent_builder{
						Name:    "dbus",
						Version: "1:1.12.20-6.el9.x86_64",
						Vulns: []*storage.EmbeddedVulnerability{
							storage.EmbeddedVulnerability_builder{
								Cve:      "CVE-2022-42010",
								Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
							}.Build(),
						},
					}.Build(),
					storage.EmbeddedImageScanComponent_builder{
						Name:    "dbus-common",
						Version: "1:1.12.20-6.el9.noarch",
						Vulns: []*storage.EmbeddedVulnerability{
							storage.EmbeddedVulnerability_builder{
								Cve:      "CVE-2022-42010",
								Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
							}.Build(),
						},
					}.Build(),
					storage.EmbeddedImageScanComponent_builder{
						Name:    "dbus-libs",
						Version: "1:1.12.20-6.el9.x86_64",
						Vulns: []*storage.EmbeddedVulnerability{
							storage.EmbeddedVulnerability_builder{
								Cve:      "CVE-2022-42010",
								Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
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
			scan: storage.ImageScan_builder{
				Components: []*storage.EmbeddedImageScanComponent{
					storage.EmbeddedImageScanComponent_builder{
						Name:    "componentD",
						Version: "1.0.0-1",
						Vulns: []*storage.EmbeddedVulnerability{
							storage.EmbeddedVulnerability_builder{
								Cve:      "CVE-TEST-10",
								Summary:  "CVE Test 10",
								Link:     "cve-link-10",
								Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
								FixedBy:  proto.String("3.0"),
							}.Build(),
						},
						FixedBy: "3.0",
					}.Build(),
					storage.EmbeddedImageScanComponent_builder{
						Name:    "componentA",
						Version: "1.0.0-1",
						Vulns:   vulnsWithAllSeverities,
						FixedBy: "2.0",
					}.Build(),
					storage.EmbeddedImageScanComponent_builder{
						Name:    "componentC",
						Version: "1.0.0-3",
						FixedBy: "2.0",
						Vulns:   vulnsWithAllSeverities,
					}.Build(),
					storage.EmbeddedImageScanComponent_builder{
						Name:    "componentB",
						Version: "1.0.0-2",
						Vulns:   vulnsWithAllSeverities,
						FixedBy: "2.0",
					}.Build(),
				},
			}.Build(),
			expectedOutput: &cveJSONResult{
				Result: cveJSONStructure{
					Summary: map[string]int{
						"TOTAL-VULNERABILITIES": 7,
						"TOTAL-COMPONENTS":      4,
						"LOW":                   2,
						"MODERATE":              1,
						"IMPORTANT":             2,
						"CRITICAL":              2,
					},
					Vulnerabilities: append(expectedVulnsComponentA, append(expectedVulnsComponentB, append(expectedVulnsComponentC, expectedVulnsComponentD...)...)...),
				},
			},
		},
		"components with vulnerabilities of all severity but filtering": {
			severities: []string{moderateCVESeverity.String(), importantCVESeverity.String()},
			scan: storage.ImageScan_builder{
				Components: []*storage.EmbeddedImageScanComponent{
					storage.EmbeddedImageScanComponent_builder{
						Name:    "componentD",
						Version: "1.0.0-1",
						Vulns: []*storage.EmbeddedVulnerability{
							storage.EmbeddedVulnerability_builder{
								Cve:      "CVE-TEST-10",
								Summary:  "CVE Test 10",
								Link:     "cve-link-10",
								Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
								FixedBy:  proto.String("3.0"),
							}.Build(),
						},
						FixedBy: "3.0",
					}.Build(),
					storage.EmbeddedImageScanComponent_builder{
						Name:    "componentA",
						Version: "1.0.0-1",
						Vulns:   vulnsWithAllSeverities,
						FixedBy: "2.0",
					}.Build(),
					storage.EmbeddedImageScanComponent_builder{
						Name:    "componentC",
						Version: "1.0.0-3",
						FixedBy: "2.0",
						Vulns:   vulnsWithAllSeverities,
					}.Build(),
					storage.EmbeddedImageScanComponent_builder{
						Name:    "componentB",
						Version: "1.0.0-2",
						Vulns:   vulnsWithAllSeverities,
						FixedBy: "2.0",
					}.Build(),
				},
			}.Build(),
			expectedOutput: &cveJSONResult{
				Result: cveJSONStructure{
					Summary: map[string]int{
						"TOTAL-VULNERABILITIES": 3,
						"TOTAL-COMPONENTS":      3,
						"LOW":                   0,
						"MODERATE":              1,
						"IMPORTANT":             2,
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
