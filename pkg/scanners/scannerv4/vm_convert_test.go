package scannerv4

import (
	"fmt"
	"testing"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testCVE1CVSSV3 = &storage.CVSSV3{
		Vector:              "CVSS:3.0/AV:N/AC:H/PR:L/UI:N/S:U/C:L/I:N/A:N",
		Score:               3.1,
		ExploitabilityScore: 1.6,
		ImpactScore:         1.4,
		AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
		AttackComplexity:    storage.CVSSV3_COMPLEXITY_HIGH,
		PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_LOW,
		UserInteraction:     storage.CVSSV3_UI_NONE,
		Scope:               storage.CVSSV3_UNCHANGED,
		Confidentiality:     storage.CVSSV3_IMPACT_LOW,
		Integrity:           storage.CVSSV3_IMPACT_NONE,
		Availability:        storage.CVSSV3_IMPACT_NONE,
		Severity:            storage.CVSSV3_LOW,
	}

	testCVE2CVSSV3 = &storage.CVSSV3{
		Vector:              "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H",
		ExploitabilityScore: 2.8,
		ImpactScore:         5.9,
		AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
		AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
		PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_NONE,
		UserInteraction:     storage.CVSSV3_UI_REQUIRED,
		Scope:               storage.CVSSV3_UNCHANGED,
		Confidentiality:     storage.CVSSV3_IMPACT_HIGH,
		Integrity:           storage.CVSSV3_IMPACT_HIGH,
		Availability:        storage.CVSSV3_IMPACT_HIGH,
		Score:               8.8,
		Severity:            storage.CVSSV3_HIGH,
	}

	testCVE3CVSSV3 = &storage.CVSSV3{
		Vector:              "CVSS:3.0/AV:A/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
		ExploitabilityScore: 2.8,
		ImpactScore:         5.9,
		AttackVector:        storage.CVSSV3_ATTACK_ADJACENT,
		AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
		PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_NONE,
		UserInteraction:     storage.CVSSV3_UI_NONE,
		Scope:               storage.CVSSV3_UNCHANGED,
		Confidentiality:     storage.CVSSV3_IMPACT_HIGH,
		Integrity:           storage.CVSSV3_IMPACT_HIGH,
		Availability:        storage.CVSSV3_IMPACT_HIGH,
		Score:               9.8,
		Severity:            storage.CVSSV3_CRITICAL,
	}
)

func TestNoVMConversionPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		var nilReport *v4.VulnerabilityReport
		_ = ToVirtualMachineScan(nilReport)

		report := &v4.VulnerabilityReport{}
		_ = ToVirtualMachineScan(report)

		report.Contents = &v4.Contents{}
		_ = ToVirtualMachineScan(report)

		report.Contents.Packages = map[string]*v4.Package{}
		_ = ToVirtualMachineScan(report)
		report.Contents.Packages["1"] = &v4.Package{Id: "1"}
		_ = ToVirtualMachineScan(report)

		report.Contents.PackagesDEPRECATED = []*v4.Package{}
		_ = ToVirtualMachineScan(report)
		report.Contents.PackagesDEPRECATED = append(report.Contents.PackagesDEPRECATED, &v4.Package{
			Id: "1",
		})
		_ = ToVirtualMachineScan(report)

		report.PackageVulnerabilities = map[string]*v4.StringList{}
		_ = ToVirtualMachineScan(report)

		report.PackageVulnerabilities["1"] = &v4.StringList{}
		_ = ToVirtualMachineScan(report)

		report.PackageVulnerabilities["1"].Values = []string{}
		_ = ToVirtualMachineScan(report)

		report.PackageVulnerabilities["1"].Values = []string{"CVE1"}
		_ = ToVirtualMachineScan(report)
	})
}

func TestToVirtualMachineScan(t *testing.T) {
	testcases := []struct {
		name     string
		contents *v4.Contents
	}{
		{
			name: "basic",
			contents: &v4.Contents{
				Packages: map[string]*v4.Package{
					"1": {
						Id:      "1",
						Name:    "my-test-package",
						Version: "1.2.3",
					},
				},
			},
		},
		{
			name: "deprecated",
			contents: &v4.Contents{
				PackagesDEPRECATED: []*v4.Package{
					{
						Id:      "1",
						Name:    "my-test-package",
						Version: "1.2.3",
					},
				},
			},
		},
	}
	input := &v4.VulnerabilityReport{
		Vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
			"CVE1-ID": {
				Id:                 "CVE1-ID",
				Name:               "CVE1-Name",
				FixedInVersion:     "v99",
				NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_IMPORTANT,
			},
		},
		PackageVulnerabilities: map[string]*v4.StringList{
			"1": {
				Values: []string{"CVE1-ID"},
			},
		},
		Notes: []v4.VulnerabilityReport_Note{
			v4.VulnerabilityReport_NOTE_OS_UNKNOWN,
		},
	}

	expected := &storage.VirtualMachineScan{
		Notes: []storage.VirtualMachineScan_Note{
			storage.VirtualMachineScan_OS_UNKNOWN,
		},
		Components: []*storage.EmbeddedVirtualMachineScanComponent{
			{
				Name:    "my-test-package",
				Version: "1.2.3",
				Vulnerabilities: []*storage.VirtualMachineVulnerability{
					{
						CveBaseInfo: &storage.VirtualMachineCVEInfo{
							Cve: "CVE1-Name",
						},
						Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
						SetFixedBy: &storage.VirtualMachineVulnerability_FixedBy{
							FixedBy: "v99",
						},
					},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			input.Contents = tc.contents
			actual := ToVirtualMachineScan(input)
			protoassert.ElementsMatch(t, expected.Components, actual.Components)
			assert.Equal(t, expected.OperatingSystem, actual.OperatingSystem)
			assert.Equal(t, expected.Notes, actual.Notes)
		})
	}
}

func TestToVirtualMachineScanNotes(t *testing.T) {
	tests := []struct {
		name     string
		input    []v4.VulnerabilityReport_Note
		expected []storage.VirtualMachineScan_Note
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: []storage.VirtualMachineScan_Note{},
		},
		{
			name:     "empty input",
			input:    []v4.VulnerabilityReport_Note{},
			expected: []storage.VirtualMachineScan_Note{},
		},
		{
			name: "mix of values",
			input: []v4.VulnerabilityReport_Note{
				v4.VulnerabilityReport_Note(-1),
				v4.VulnerabilityReport_NOTE_OS_UNSUPPORTED,
				v4.VulnerabilityReport_NOTE_OS_UNKNOWN,
				v4.VulnerabilityReport_NOTE_UNSPECIFIED,
			},
			expected: []storage.VirtualMachineScan_Note{
				storage.VirtualMachineScan_UNSET,
				storage.VirtualMachineScan_OS_UNSUPPORTED,
				storage.VirtualMachineScan_OS_UNKNOWN,
				storage.VirtualMachineScan_UNSET,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(it *testing.T) {
			actual := toVirtualMachineScanNotes(tc.input)
			assert.Equal(it, tc.expected, actual)
		})
	}
}

func TestToVirtualMachineScanComponents(t *testing.T) {
	tests := []struct {
		name     string
		report   *v4.VulnerabilityReport
		expected []*storage.EmbeddedVirtualMachineScanComponent
	}{
		{
			name:     "nil input",
			report:   nil,
			expected: []*storage.EmbeddedVirtualMachineScanComponent{},
		},
		{
			name: "basic, no vulnerabilities",
			report: &v4.VulnerabilityReport{
				Contents: &v4.Contents{
					Packages: map[string]*v4.Package{
						"1": {
							Id:      "1",
							Name:    "glib2",
							Version: "2.68.4-14.el9",
						},
					},
				},
			},
			expected: []*storage.EmbeddedVirtualMachineScanComponent{
				{
					Name:    "glib2",
					Version: "2.68.4-14.el9",
				},
			},
		},
		{
			name: "basic, deprecated no vulnerabilities",
			report: &v4.VulnerabilityReport{
				Contents: &v4.Contents{
					PackagesDEPRECATED: []*v4.Package{
						{
							Id:      "1",
							Name:    "glib2",
							Version: "2.68.4-14.el9",
						},
					},
				},
			},
			expected: []*storage.EmbeddedVirtualMachineScanComponent{
				{
					Name:    "glib2",
					Version: "2.68.4-14.el9",
				},
			},
		},
		{
			name: "basic, with matching vulnerabilities",
			report: &v4.VulnerabilityReport{
				Contents: &v4.Contents{
					Packages: map[string]*v4.Package{
						"1": {
							Id:      "1",
							Name:    "glib2",
							Version: "2.68.4-14.el9",
						},
						"2": {
							Id:      "2",
							Name:    "postgres",
							Version: "15.10",
						},
					},
				},
				PackageVulnerabilities: map[string]*v4.StringList{
					"2": {
						Values: []string{"CVE-2025-8715-ID"},
					},
				},
				Vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
					"CVE-2025-8715-ID": {
						Id:          "CVE-2025-8715-ID",
						Name:        "CVE-2025-8715",
						Description: "some vulnerability description",
					},
				},
			},
			expected: []*storage.EmbeddedVirtualMachineScanComponent{
				{
					Name:    "glib2",
					Version: "2.68.4-14.el9",
				},
				{
					Name:    "postgres",
					Version: "15.10",
					Vulnerabilities: []*storage.VirtualMachineVulnerability{
						{
							CveBaseInfo: &storage.VirtualMachineCVEInfo{
								Cve:     "CVE-2025-8715",
								Summary: "some vulnerability description",
							},
						},
					},
				},
			},
		},
		{
			name: "basic, deprecated with matching vulnerabilities",
			report: &v4.VulnerabilityReport{
				Contents: &v4.Contents{
					PackagesDEPRECATED: []*v4.Package{
						{
							Id:      "1",
							Name:    "glib2",
							Version: "2.68.4-14.el9",
						},
						{
							Id:      "2",
							Name:    "postgres",
							Version: "15.10",
						},
					},
				},
				PackageVulnerabilities: map[string]*v4.StringList{
					"2": {
						Values: []string{"CVE-2025-8715-ID"},
					},
				},
				Vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
					"CVE-2025-8715-ID": {
						Id:          "CVE-2025-8715-ID",
						Name:        "CVE-2025-8715",
						Description: "some vulnerability description",
					},
				},
			},
			expected: []*storage.EmbeddedVirtualMachineScanComponent{
				{
					Name:    "glib2",
					Version: "2.68.4-14.el9",
				},
				{
					Name:    "postgres",
					Version: "15.10",
					Vulnerabilities: []*storage.VirtualMachineVulnerability{
						{
							CveBaseInfo: &storage.VirtualMachineCVEInfo{
								Cve:     "CVE-2025-8715",
								Summary: "some vulnerability description",
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(it *testing.T) {
			actual := toVirtualMachineComponents(tc.report)
			protoassert.ElementsMatch(it, tc.expected, actual)
		})
	}
}

func TestToVirtualMachineScanComponentVulnerabilities(t *testing.T) {
	tests := []struct {
		name                 string
		vulnerabilitiesByIDs map[string]*v4.VulnerabilityReport_Vulnerability
		vulnerabilityIDs     []string
		expected             []*storage.VirtualMachineVulnerability
	}{
		{
			name:                 "nil inputs",
			vulnerabilitiesByIDs: nil,
			vulnerabilityIDs:     nil,
			expected:             []*storage.VirtualMachineVulnerability{},
		},
		{
			name:                 "nil mapping, empty IDs",
			vulnerabilitiesByIDs: nil,
			vulnerabilityIDs:     []string{},
			expected:             []*storage.VirtualMachineVulnerability{},
		},
		{
			name:                 "empty mapping, nil IDs",
			vulnerabilitiesByIDs: map[string]*v4.VulnerabilityReport_Vulnerability{},
			vulnerabilityIDs:     nil,
			expected:             []*storage.VirtualMachineVulnerability{},
		},
		{
			name:                 "empty mapping, empty IDs",
			vulnerabilitiesByIDs: map[string]*v4.VulnerabilityReport_Vulnerability{},
			vulnerabilityIDs:     []string{},
			expected:             []*storage.VirtualMachineVulnerability{},
		},
		{
			name: "non-empty mapping, empty IDs",
			vulnerabilitiesByIDs: map[string]*v4.VulnerabilityReport_Vulnerability{
				"CVE-2025-8715-ID": {
					Id:          "CVE-2025-8715-ID",
					Name:        "CVE-2025-8715",
					Description: "some vulnerability description",
				},
			},
			vulnerabilityIDs: []string{},
			expected:         []*storage.VirtualMachineVulnerability{},
		},
		{
			name: "ID not in mapping",
			vulnerabilitiesByIDs: map[string]*v4.VulnerabilityReport_Vulnerability{
				"CVE-2025-8715-ID": {
					Id:          "CVE-2025-8715-ID",
					Name:        "CVE-2025-8715",
					Description: "some vulnerability description",
				},
			},
			vulnerabilityIDs: []string{"CVE-2025-8714-ID"},
			expected:         []*storage.VirtualMachineVulnerability{},
		},
		{
			name: "duplicate component vulnerability ID",
			vulnerabilitiesByIDs: map[string]*v4.VulnerabilityReport_Vulnerability{
				"CVE-2025-8713-ID": {
					Id:          "CVE-2025-8713-ID",
					Name:        "CVE-2025-8713",
					Description: "some vulnerability description",
					CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
						{
							V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
								BaseScore: 3.1,
								Vector:    "CVSS:3.0/AV:N/AC:H/PR:L/UI:N/S:U/C:L/I:N/A:N",
							},
						},
					},
				},
			},
			vulnerabilityIDs: []string{"CVE-2025-8713-ID", "CVE-2025-8713-ID"},
			expected: []*storage.VirtualMachineVulnerability{
				{
					CveBaseInfo: &storage.VirtualMachineCVEInfo{
						Cve:     "CVE-2025-8713",
						Summary: "some vulnerability description",
						CvssMetrics: []*storage.CVSSScore{
							{
								CvssScore: &storage.CVSSScore_Cvssv3{
									Cvssv3: testCVE1CVSSV3,
								},
							},
						},
					},
					Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
				},
			},
		},
		{
			name: "multiple mapping matches, some misses (silently skipped)",
			vulnerabilitiesByIDs: map[string]*v4.VulnerabilityReport_Vulnerability{
				"CVE-2025-8715-ID": {
					Id:          "CVE-2025-8715-ID",
					Name:        "CVE-2025-8715",
					Description: "some vulnerability description",
				},
				"CVE-2025-8713-ID": {
					Id:          "CVE-2025-8713-ID",
					Name:        "CVE-2025-8713",
					Description: "some other vulnerability description",
					CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
						{
							V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
								BaseScore: 3.1,
								Vector:    "CVSS:3.0/AV:N/AC:H/PR:L/UI:N/S:U/C:L/I:N/A:N",
							},
						},
					},
				},
			},
			vulnerabilityIDs: []string{"CVE-2025-4207-ID", "CVE-2025-8713-ID", "CVE-2025-8714-ID", "CVE-2025-8715-ID"},
			expected: []*storage.VirtualMachineVulnerability{
				{
					CveBaseInfo: &storage.VirtualMachineCVEInfo{
						Cve:     "CVE-2025-8713",
						Summary: "some other vulnerability description",
						CvssMetrics: []*storage.CVSSScore{
							{
								CvssScore: &storage.CVSSScore_Cvssv3{
									Cvssv3: testCVE1CVSSV3,
								},
							},
						},
					},
					Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
				},
				{
					CveBaseInfo: &storage.VirtualMachineCVEInfo{
						Cve:     "CVE-2025-8715",
						Summary: "some vulnerability description",
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(it *testing.T) {
			actual := toVirtualMachineScanComponentVulnerabilities(tc.vulnerabilitiesByIDs, tc.vulnerabilityIDs)
			protoassert.SlicesEqual(it, tc.expected, actual)
		})
	}
}

func TestToVirtualMachineScanComponentVulnerabilitiesScoringError(t *testing.T) {
	vulnerabilitiesByIDs := map[string]*v4.VulnerabilityReport_Vulnerability{
		"CVE-2025-8713-ID": {
			Id:          "CVE-2025-8713-ID",
			Name:        "CVE-2025-8713",
			Description: "some vulnerability description",
			CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
				{
					V2: &v4.VulnerabilityReport_Vulnerability_CVSS_V2{
						BaseScore: 3.1,
						Vector:    "CVSS:3.0/AV:N/AC:H/PR:L/UI:N/S:U/C:L/I:N/A:N",
					},
				},
			},
		},
	}
	vulnerabilityIDs := []string{"CVE-2025-8713-ID"}

	expected := []*storage.VirtualMachineVulnerability{
		{
			CveBaseInfo: &storage.VirtualMachineCVEInfo{
				Cve:     "CVE-2025-8713",
				Summary: "some vulnerability description",
			},
		},
	}

	if buildinfo.ReleaseBuild {
		t.Log("Release")
		actual := toVirtualMachineScanComponentVulnerabilities(vulnerabilitiesByIDs, vulnerabilityIDs)
		protoassert.SlicesEqual(t, expected, actual)
	} else {
		t.Log("Debug")
		assert.Panics(t, func() {
			toVirtualMachineScanComponentVulnerabilities(vulnerabilitiesByIDs, vulnerabilityIDs)
		})
	}
}

func TestSetVirtualMachineScoresAndScoreVersions(t *testing.T) {
	badInputTests := []struct {
		name          string
		vulnerability *storage.VirtualMachineVulnerability
		cvssMetrics   []*v4.VulnerabilityReport_Vulnerability_CVSS
		expected      *storage.VirtualMachineVulnerability
		expectedError error
	}{
		{
			name:          "nil input vulnerability yields error",
			vulnerability: nil,
			cvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
				{
					V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
						BaseScore: 3.1,
						Vector:    "CVSS:3.0/AV:N/AC:H/PR:L/UI:N/S:U/C:L/I:N/A:N",
					},
				},
			},
			expected:      nil,
			expectedError: errox.InvalidArgs,
		},
		{
			name:          "input vulnerability with nil CveBaseInfo yields error",
			vulnerability: &storage.VirtualMachineVulnerability{},
			cvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
				{
					V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
						BaseScore: 3.1,
						Vector:    "CVSS:3.0/AV:N/AC:H/PR:L/UI:N/S:U/C:L/I:N/A:N",
					},
				},
			},
			expected:      &storage.VirtualMachineVulnerability{},
			expectedError: errox.InvalidArgs,
		},
		{
			name: "valid input vulnerability and nil metrics result in no error but no update",
			vulnerability: &storage.VirtualMachineVulnerability{
				CveBaseInfo: &storage.VirtualMachineCVEInfo{
					Cve: "CVE-2025-8715",
				},
			},
			cvssMetrics: nil,
			expected: &storage.VirtualMachineVulnerability{
				CveBaseInfo: &storage.VirtualMachineCVEInfo{
					Cve: "CVE-2025-8715",
				},
			},
			expectedError: nil,
		},
		{
			name: "valid input vulnerability and empty metrics result in no error but no update",
			vulnerability: &storage.VirtualMachineVulnerability{
				CveBaseInfo: &storage.VirtualMachineCVEInfo{
					Cve: "CVE-2025-8715",
				},
			},
			cvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{},
			expected: &storage.VirtualMachineVulnerability{
				CveBaseInfo: &storage.VirtualMachineCVEInfo{
					Cve: "CVE-2025-8715",
				},
			},
			expectedError: nil,
		},
	}

	for _, tc := range badInputTests {
		t.Run(fmt.Sprintf("bad input/%s", tc.name), func(it *testing.T) {
			input := tc.vulnerability.CloneVT()
			err := setVirtualMachineScoresAndScoreVersions(input, tc.cvssMetrics)
			if tc.expectedError != nil {
				assert.ErrorIs(it, err, tc.expectedError)
			} else {
				assert.NoError(it, err)
			}
			protoassert.Equal(it, tc.expected, input)
		})
	}

	expectedCVSSV3 := testCVE2CVSSV3

	expectedFakeNVDCVSSV3 := testCVE3CVSSV3

	validInputTests := []struct {
		name          string
		vulnerability *storage.VirtualMachineVulnerability
		cvssMetrics   []*v4.VulnerabilityReport_Vulnerability_CVSS
		expected      *storage.VirtualMachineVulnerability
	}{
		{
			name: "basic update",
			vulnerability: &storage.VirtualMachineVulnerability{
				CveBaseInfo: &storage.VirtualMachineCVEInfo{
					Cve: "CVE-2025-8715",
				},
			},
			cvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
				{
					V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
						BaseScore: 8.8,
						Vector:    "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H",
					},
				},
			},
			expected: &storage.VirtualMachineVulnerability{
				CveBaseInfo: &storage.VirtualMachineCVEInfo{
					Cve: "CVE-2025-8715",
					CvssMetrics: []*storage.CVSSScore{
						{
							CvssScore: &storage.CVSSScore_Cvssv3{
								Cvssv3: expectedCVSSV3,
							},
						},
					},
				},
				Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
			},
		},
		{
			name: "CVSS V2 is only propagated to CVSS metrics in enriched structure",
			vulnerability: &storage.VirtualMachineVulnerability{
				CveBaseInfo: &storage.VirtualMachineCVEInfo{
					Cve: "CVE-2025-8715",
				},
			},
			cvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
				{
					V2: &v4.VulnerabilityReport_Vulnerability_CVSS_V2{
						BaseScore: 9.0,
						Vector:    "AV:N/AC:L/Au:S/C:C/I:C/A:C",
					},
				},
			},
			expected: &storage.VirtualMachineVulnerability{
				CveBaseInfo: &storage.VirtualMachineCVEInfo{
					Cve: "CVE-2025-8715",
					CvssMetrics: []*storage.CVSSScore{
						{
							CvssScore: &storage.CVSSScore_Cvssv2{
								Cvssv2: &storage.CVSSV2{
									Vector:              "AV:N/AC:L/Au:S/C:C/I:C/A:C",
									ExploitabilityScore: 8.0,
									ImpactScore:         10.0,
									AttackVector:        storage.CVSSV2_ATTACK_NETWORK,
									AccessComplexity:    storage.CVSSV2_ACCESS_LOW,
									Authentication:      storage.CVSSV2_AUTH_SINGLE,
									Confidentiality:     storage.CVSSV2_IMPACT_COMPLETE,
									Integrity:           storage.CVSSV2_IMPACT_COMPLETE,
									Availability:        storage.CVSSV2_IMPACT_COMPLETE,
									Score:               9.0,
									Severity:            storage.CVSSV2_HIGH,
								},
							},
						},
					},
				},
				Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
			},
		},
		{
			name: "CVSS V3 data from NVD is considered if it is the only input",
			vulnerability: &storage.VirtualMachineVulnerability{
				CveBaseInfo: &storage.VirtualMachineCVEInfo{
					Cve: "CVE-2025-8715",
				},
			},
			cvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
				{
					Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
					V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
						BaseScore: 8.8,
						Vector:    "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H",
					},
				},
			},
			expected: &storage.VirtualMachineVulnerability{
				CveBaseInfo: &storage.VirtualMachineCVEInfo{
					Cve: "CVE-2025-8715",
					CvssMetrics: []*storage.CVSSScore{
						{
							Source: storage.Source_SOURCE_NVD,
							CvssScore: &storage.CVSSScore_Cvssv3{
								Cvssv3: expectedCVSSV3,
							},
						},
					},
				},
				Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
			},
		},
		{
			name: "CVSS V3 data from NVD is ignored in favour of other CVSS info (only propagated in CVSSMetrics) if it is NOT the only input (first position)",
			vulnerability: &storage.VirtualMachineVulnerability{
				CveBaseInfo: &storage.VirtualMachineCVEInfo{
					Cve: "CVE-2025-8715",
				},
			},
			cvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
				{
					Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
					V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
						BaseScore: 9.8,
						Vector:    "CVSS:3.0/AV:A/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
					},
				},
				{
					Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_OSV,
					V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
						BaseScore: 8.8,
						Vector:    "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H",
					},
				},
			},
			expected: &storage.VirtualMachineVulnerability{
				CveBaseInfo: &storage.VirtualMachineCVEInfo{
					Cve: "CVE-2025-8715",
					CvssMetrics: []*storage.CVSSScore{
						{
							Source: storage.Source_SOURCE_NVD,
							CvssScore: &storage.CVSSScore_Cvssv3{
								Cvssv3: expectedFakeNVDCVSSV3,
							},
						},
						{
							Source: storage.Source_SOURCE_OSV,
							CvssScore: &storage.CVSSScore_Cvssv3{
								Cvssv3: expectedCVSSV3,
							},
						},
					},
				},
				Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
			},
		},
		{
			name: "CVSS V3 data from NVD is ignored in favour of other CVSS info (only propagated in CVSSMetrics) if it is NOT the only input (second position)",
			vulnerability: &storage.VirtualMachineVulnerability{
				CveBaseInfo: &storage.VirtualMachineCVEInfo{
					Cve: "CVE-2025-8715",
				},
			},
			cvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
				{
					Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_RED_HAT,
					V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
						BaseScore: 8.8,
						Vector:    "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H",
					},
				},
				{
					Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
					V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
						BaseScore: 9.8,
						Vector:    "CVSS:3.0/AV:A/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
					},
				},
			},
			expected: &storage.VirtualMachineVulnerability{
				CveBaseInfo: &storage.VirtualMachineCVEInfo{
					Cve: "CVE-2025-8715",
					CvssMetrics: []*storage.CVSSScore{
						{
							Source: storage.Source_SOURCE_RED_HAT,
							CvssScore: &storage.CVSSScore_Cvssv3{
								Cvssv3: expectedCVSSV3,
							},
						},
						{
							Source: storage.Source_SOURCE_NVD,
							CvssScore: &storage.CVSSScore_Cvssv3{
								Cvssv3: expectedFakeNVDCVSSV3,
							},
						},
					},
				},
				Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
			},
		},
	}

	for _, tc := range validInputTests {
		t.Run(fmt.Sprintf("valid input/%s", tc.name), func(it *testing.T) {
			input := tc.vulnerability.CloneVT()
			err := setVirtualMachineScoresAndScoreVersions(input, tc.cvssMetrics)
			assert.NoError(it, err)
			protoassert.Equal(it, tc.expected, input)
			for ix, expectedMetric := range tc.expected.GetCveBaseInfo().GetCvssMetrics() {
				require.Less(it, ix, len(input.GetCveBaseInfo().GetCvssMetrics()))
				protoassert.Equal(it, expectedMetric, input.GetCveBaseInfo().GetCvssMetrics()[ix])
			}
		})
	}
}

func TestSetVirtualMachineScoresAndScoreVersionsParseErrors(t *testing.T) {
	tests := map[string]struct {
		cvssMetrics      []*v4.VulnerabilityReport_Vulnerability_CVSS
		expectedErrorMsg string
	}{
		"bad CVSSV2 vector": {
			cvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
				{
					V2: &v4.VulnerabilityReport_Vulnerability_CVSS_V2{
						BaseScore: 8.8,
						Vector:    "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H",
					},
				},
			},
			expectedErrorMsg: "failed to get CVSS metrics error: parsing CVSS v2 vector: invalid CVSSv2 vector \"CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H\": undefined metric CVSS with value 3.0",
		},
		"bad CVSSV3 vector": {
			cvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
				{
					V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
						BaseScore: 8.8,
						Vector:    "AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H",
					},
				},
			},
			expectedErrorMsg: "failed to get CVSS metrics error: parsing CVSS v3 vector: invalid CVSSv3 vector \"AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H\": vector missing \"CVSS:\" prefix: \"AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H\"",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(it *testing.T) {
			vuln := &storage.VirtualMachineVulnerability{
				CveBaseInfo: &storage.VirtualMachineCVEInfo{
					Cve: "some-CVE",
				},
			}
			ioVuln := vuln.CloneVT()
			err := setVirtualMachineScoresAndScoreVersions(ioVuln, tc.cvssMetrics)
			assert.ErrorContains(it, err, tc.expectedErrorMsg)
		})
	}
}
