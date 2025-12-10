package scannerv4

import (
	"testing"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
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
				Notes: []storage.EmbeddedVirtualMachineScanComponent_Note{
					storage.EmbeddedVirtualMachineScanComponent_UNSCANNED,
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			input.Contents = tc.contents
			actual := ToVirtualMachineScan(input)
			protoassert.ElementsMatch(t, expected.GetComponents(), actual.GetComponents())
			assert.Equal(t, expected.GetOperatingSystem(), actual.GetOperatingSystem())
			assert.Equal(t, expected.GetNotes(), actual.GetNotes())
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
					Notes: []storage.EmbeddedVirtualMachineScanComponent_Note{
						storage.EmbeddedVirtualMachineScanComponent_UNSCANNED,
					},
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
					Notes: []storage.EmbeddedVirtualMachineScanComponent_Note{
						storage.EmbeddedVirtualMachineScanComponent_UNSCANNED,
					},
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
					Notes: []storage.EmbeddedVirtualMachineScanComponent_Note{
						storage.EmbeddedVirtualMachineScanComponent_UNSCANNED,
					},
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
					Notes: []storage.EmbeddedVirtualMachineScanComponent_Note{
						storage.EmbeddedVirtualMachineScanComponent_UNSCANNED,
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
					Notes: []storage.EmbeddedVirtualMachineScanComponent_Note{
						storage.EmbeddedVirtualMachineScanComponent_UNSCANNED,
					},
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
					Notes: []storage.EmbeddedVirtualMachineScanComponent_Note{
						storage.EmbeddedVirtualMachineScanComponent_UNSCANNED,
					},
				},
			},
		},
		{
			name: "scan component with valid CPE",
			report: &v4.VulnerabilityReport{
				Contents: &v4.Contents{
					Packages: map[string]*v4.Package{
						"1": {
							Id:      "1",
							Name:    "my-test-package",
							Version: "1.2.3",
						},
					},
					Repositories: map[string]*v4.Repository{
						"rhel-9-for-x86_64-appstream-rpms": {
							Id:   "rhel-9-for-x86_64-appstream-rpms",
							Name: "rhel-9-for-x86_64-appstream-rpms",
							Cpe:  "cpe:2.3:a:redhat:enterprise_linux:9:*:appstream:*:*:*:*:*",
						},
					},
					Environments: map[string]*v4.Environment_List{
						"1": {
							Environments: []*v4.Environment{
								{
									PackageDb: "sqlite:var/lib/rpm",
									RepositoryIds: []string{
										"rhel-9-for-x86_64-appstream-rpms",
										"rhel-9-for-x86_64-baseos-rpms",
									},
								},
							},
						},
					},
				},
			},
			expected: []*storage.EmbeddedVirtualMachineScanComponent{
				{
					Name:    "my-test-package",
					Version: "1.2.3",
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
					EpssMetrics: &v4.VulnerabilityReport_Vulnerability_EPSS{
						Probability: .42,
						Percentile:  .84,
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
						Epss: &storage.VirtualMachineEPSS{
							EpssProbability: .42,
							EpssPercentile:  .84,
						},
					},
					Cvss:     3.1,
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
					Cvss:     3.1,
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
