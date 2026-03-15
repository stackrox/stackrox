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
			Cvss:       1.2,
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.0"},
		},
		{
			Cve:        "CVE-TEST-1",
			Summary:    "CVE Test 1",
			Link:       "cve-link-1",
			Severity:   storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
			Cvss:       10.0,
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.2"},
		},
		{
			Cve:        "CVE-TEST-3",
			Summary:    "CVE Test 3",
			Link:       "cve-link-3",
			Severity:   storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
			Cvss:       7.5,
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.3"},
		},
		{
			Cve:        "CVE-TEST-4",
			Summary:    "CVE Test 4",
			Link:       "cve-link-4",
			Severity:   storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.4"},
		},
		{
			Cve:        "CVE-TEST-5",
			Summary:    "CVE Test 5",
			Link:       "cve-link-5",
			Severity:   storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
			Cvss:       8.1,
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.5"},
			Advisory: &storage.Advisory{
				Name: "ADVISORY-TEST-5",
				Link: "advisory-link-5",
			},
		},
		{
			Cve:        "CVE-TEST-6",
			Summary:    "CVE Test 6",
			Link:       "cve-link-6",
			Severity:   storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.6"},
			Advisory: &storage.Advisory{
				Name: "ADVISORY-TEST-6",
				Link: "advisory-link-6",
			},
		},
	}

	// Expected vulns and components without filtering.
	expectedVulnsComponentA := []CVEVulnerabilityJSON{
		{CveID: "CVE-TEST-1", CveSeverity: CriticalCVESeverity, CveInfo: "cve-link-1", CveCVSS: 10.0, ComponentName: "componentA", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "1.2"},
		{CveID: "CVE-TEST-5", CveSeverity: ImportantCVESeverity, CveInfo: "cve-link-5", CveCVSS: 8.1, ComponentName: "componentA", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "1.5", AdvisoryID: "ADVISORY-TEST-5", AdvisoryInfo: "advisory-link-5"},
		{CveID: "CVE-TEST-3", CveSeverity: ImportantCVESeverity, CveInfo: "cve-link-3", CveCVSS: 7.5, ComponentName: "componentA", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "1.3"},
		{CveID: "CVE-TEST-4", CveSeverity: ModerateCVESeverity, CveInfo: "cve-link-4", ComponentName: "componentA", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "1.4"},
		{CveID: "CVE-TEST-2", CveSeverity: LowCVESeverity, CveInfo: "cve-link-2", CveCVSS: 1.2, ComponentName: "componentA", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "1.0"},
		{CveID: "CVE-TEST-6", CveSeverity: LowCVESeverity, CveInfo: "cve-link-6", ComponentName: "componentA", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "1.6", AdvisoryID: "ADVISORY-TEST-6", AdvisoryInfo: "advisory-link-6"},
	}
	expectedVulnsComponentB := []CVEVulnerabilityJSON{
		{CveID: "CVE-TEST-1", CveSeverity: CriticalCVESeverity, CveInfo: "cve-link-1", CveCVSS: 10.0, ComponentName: "componentB", ComponentVersion: "1.0.0-2", ComponentFixedVersion: "1.2"},
		{CveID: "CVE-TEST-5", CveSeverity: ImportantCVESeverity, CveInfo: "cve-link-5", CveCVSS: 8.1, ComponentName: "componentB", ComponentVersion: "1.0.0-2", ComponentFixedVersion: "1.5", AdvisoryID: "ADVISORY-TEST-5", AdvisoryInfo: "advisory-link-5"},
		{CveID: "CVE-TEST-3", CveSeverity: ImportantCVESeverity, CveInfo: "cve-link-3", CveCVSS: 7.5, ComponentName: "componentB", ComponentVersion: "1.0.0-2", ComponentFixedVersion: "1.3"},
		{CveID: "CVE-TEST-4", CveSeverity: ModerateCVESeverity, CveInfo: "cve-link-4", ComponentName: "componentB", ComponentVersion: "1.0.0-2", ComponentFixedVersion: "1.4"},
		{CveID: "CVE-TEST-2", CveSeverity: LowCVESeverity, CveInfo: "cve-link-2", CveCVSS: 1.2, ComponentName: "componentB", ComponentVersion: "1.0.0-2", ComponentFixedVersion: "1.0"},
		{CveID: "CVE-TEST-6", CveSeverity: LowCVESeverity, CveInfo: "cve-link-6", ComponentName: "componentB", ComponentVersion: "1.0.0-2", ComponentFixedVersion: "1.6", AdvisoryID: "ADVISORY-TEST-6", AdvisoryInfo: "advisory-link-6"},
	}
	expectedVulnsComponentC := []CVEVulnerabilityJSON{
		{CveID: "CVE-TEST-1", CveSeverity: CriticalCVESeverity, CveInfo: "cve-link-1", CveCVSS: 10.0, ComponentName: "componentC", ComponentVersion: "1.0.0-3", ComponentFixedVersion: "1.2"},
		{CveID: "CVE-TEST-5", CveSeverity: ImportantCVESeverity, CveInfo: "cve-link-5", CveCVSS: 8.1, ComponentName: "componentC", ComponentVersion: "1.0.0-3", ComponentFixedVersion: "1.5", AdvisoryID: "ADVISORY-TEST-5", AdvisoryInfo: "advisory-link-5"},
		{CveID: "CVE-TEST-3", CveSeverity: ImportantCVESeverity, CveInfo: "cve-link-3", CveCVSS: 7.5, ComponentName: "componentC", ComponentVersion: "1.0.0-3", ComponentFixedVersion: "1.3"},
		{CveID: "CVE-TEST-4", CveSeverity: ModerateCVESeverity, CveInfo: "cve-link-4", ComponentName: "componentC", ComponentVersion: "1.0.0-3", ComponentFixedVersion: "1.4"},
		{CveID: "CVE-TEST-2", CveSeverity: LowCVESeverity, CveInfo: "cve-link-2", CveCVSS: 1.2, ComponentName: "componentC", ComponentVersion: "1.0.0-3", ComponentFixedVersion: "1.0"},
		{CveID: "CVE-TEST-6", CveSeverity: LowCVESeverity, CveInfo: "cve-link-6", ComponentName: "componentC", ComponentVersion: "1.0.0-3", ComponentFixedVersion: "1.6", AdvisoryID: "ADVISORY-TEST-6", AdvisoryInfo: "advisory-link-6"},
	}
	expectedVulnsComponentD := []CVEVulnerabilityJSON{
		{
			CveID: "CVE-TEST-10", CveSeverity: CriticalCVESeverity, CveInfo: "cve-link-10", ComponentName: "componentD", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "3.0",
		},
	}

	// Expected vulns and components when filtered.
	expectedVulnsComponentAFiltered := []CVEVulnerabilityJSON{
		{CveID: "CVE-TEST-5", CveSeverity: ImportantCVESeverity, CveInfo: "cve-link-5", CveCVSS: 8.1, ComponentName: "componentA", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "1.5", AdvisoryID: "ADVISORY-TEST-5", AdvisoryInfo: "advisory-link-5"},
		{CveID: "CVE-TEST-3", CveSeverity: ImportantCVESeverity, CveInfo: "cve-link-3", CveCVSS: 7.5, ComponentName: "componentA", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "1.3"},
		{CveID: "CVE-TEST-4", CveSeverity: ModerateCVESeverity, CveInfo: "cve-link-4", ComponentName: "componentA", ComponentVersion: "1.0.0-1", ComponentFixedVersion: "1.4"},
	}
	expectedVulnsComponentBFiltered := []CVEVulnerabilityJSON{
		{CveID: "CVE-TEST-5", CveSeverity: ImportantCVESeverity, CveInfo: "cve-link-5", CveCVSS: 8.1, ComponentName: "componentB", ComponentVersion: "1.0.0-2", ComponentFixedVersion: "1.5", AdvisoryID: "ADVISORY-TEST-5", AdvisoryInfo: "advisory-link-5"},
		{CveID: "CVE-TEST-3", CveSeverity: ImportantCVESeverity, CveInfo: "cve-link-3", CveCVSS: 7.5, ComponentName: "componentB", ComponentVersion: "1.0.0-2", ComponentFixedVersion: "1.3"},
		{CveID: "CVE-TEST-4", CveSeverity: ModerateCVESeverity, CveInfo: "cve-link-4", ComponentName: "componentB", ComponentVersion: "1.0.0-2", ComponentFixedVersion: "1.4"},
	}
	expectedVulnsComponentCFiltered := []CVEVulnerabilityJSON{
		{CveID: "CVE-TEST-5", CveSeverity: ImportantCVESeverity, CveInfo: "cve-link-5", CveCVSS: 8.1, ComponentName: "componentC", ComponentVersion: "1.0.0-3", ComponentFixedVersion: "1.5", AdvisoryID: "ADVISORY-TEST-5", AdvisoryInfo: "advisory-link-5"},
		{CveID: "CVE-TEST-3", CveSeverity: ImportantCVESeverity, CveInfo: "cve-link-3", CveCVSS: 7.5, ComponentName: "componentC", ComponentVersion: "1.0.0-3", ComponentFixedVersion: "1.3"},
		{CveID: "CVE-TEST-4", CveSeverity: ModerateCVESeverity, CveInfo: "cve-link-4", ComponentName: "componentC", ComponentVersion: "1.0.0-3", ComponentFixedVersion: "1.4"},
	}

	cases := map[string]struct {
		scan           *storage.ImageScan
		severities     []string
		expectedOutput *CVEJSONResult
	}{
		"empty img scan results": {
			scan: &storage.ImageScan{
				Components: nil,
			},
			severities: AllSeverities(),
			expectedOutput: &CVEJSONResult{
				Result: CVEJSONStructure{
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
			severities: []string{
				LowCVESeverity.String(),
				ModerateCVESeverity.String(),
				ImportantCVESeverity.String(),
				CriticalCVESeverity.String()},
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
			expectedOutput: &CVEJSONResult{
				Result: CVEJSONStructure{
					Summary: map[string]int{
						"TOTAL-VULNERABILITIES": 1,
						"TOTAL-COMPONENTS":      3,
						"LOW":                   0,
						"MODERATE":              1,
						"IMPORTANT":             0,
						"CRITICAL":              0,
					},
					Vulnerabilities: []CVEVulnerabilityJSON{
						{
							CveID:            "CVE-2022-42010",
							CveSeverity:      ModerateCVESeverity,
							ComponentName:    "dbus",
							ComponentVersion: "1:1.12.20-6.el9.x86_64",
						},
						{
							CveID:            "CVE-2022-42010",
							CveSeverity:      ModerateCVESeverity,
							ComponentName:    "dbus-common",
							ComponentVersion: "1:1.12.20-6.el9.noarch",
						},
						{
							CveID:            "CVE-2022-42010",
							CveSeverity:      ModerateCVESeverity,
							ComponentName:    "dbus-libs",
							ComponentVersion: "1:1.12.20-6.el9.x86_64",
						},
					},
				},
			},
		},
		"components with vulnerabilities of all severity": {
			severities: []string{
				LowCVESeverity.String(),
				ModerateCVESeverity.String(),
				ImportantCVESeverity.String(),
				CriticalCVESeverity.String()},
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
			expectedOutput: &CVEJSONResult{
				Result: CVEJSONStructure{
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
			severities: []string{
				ModerateCVESeverity.String(),
				ImportantCVESeverity.String()},
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
			expectedOutput: &CVEJSONResult{
				Result: CVEJSONStructure{
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
			cveSummary := NewCVESummaryForPrinting(c.scan, c.severities)
			assert.Equal(t, c.expectedOutput, cveSummary)
		})
	}
}
