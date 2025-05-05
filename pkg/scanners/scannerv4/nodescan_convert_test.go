package scannerv4

import (
	"testing"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
)

var (
	mockVulnReport = &v4.VulnerabilityReport{
		HashId: "",
		Vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
			"7401229": {
				Id:                 "7401229",
				Name:               "RHSA-2024:4616",
				Description:        "Sample Description",
				Severity:           "Moderate",
				NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_MODERATE,
				FixedInVersion:     "0:4.16.0-202407111006.p0.gfa84651.assembly.stream.el9",
				Link:               "https://localhost/7401229",
				Cvss: &v4.VulnerabilityReport_Vulnerability_CVSS{
					V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
						BaseScore: 7.5,
						Vector:    "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H",
					},
					Url: "https://localhost/7401229",
				},
			},
		},
		PackageVulnerabilities: map[string]*v4.StringList{
			"0": {
				Values: []string{"7401229"},
			},
		},
		Contents: &v4.Contents{
			Packages: []*v4.Package{
				{
					Id:      "0",
					Name:    "openssh-clients",
					Version: "8.7p1-38.el9",
					Kind:    "binary",
					Source: &v4.Package{
						Name:    "openssh",
						Version: "8.7p1-38.el9",
						Kind:    "source",
						Source:  nil,
						Cpe:     "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
					},
					PackageDb:      "sqlite:usr/share/rpm",
					RepositoryHint: "hash:sha256:f52ca767328e6919ec11a1da654e92743587bd3c008f0731f8c4de3af19c1830|key:199e2f91fd431d51",
					Arch:           "x86_64",
					Cpe:            "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
				},
				{
					Id:      "1",
					Name:    "skopeo",
					Version: "2:1.14.4-2.rhaos4.16.el9",
					Kind:    "binary",
					Source: &v4.Package{
						Name:    "skopeo",
						Version: "2:1.14.4-2.rhaos4.16.el9",
						Kind:    "source",
						Cpe:     "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
					},
					PackageDb:      "sqlite:usr/share/rpm",
					RepositoryHint: "hash:sha256:072a75d1b9b36457751ef05031fd69615f21ebaa935c30d74d827328b78fa694|key:199e2f91fd431d51",
					Arch:           "x86_64",
					Cpe:            "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
				},
			},
			Repositories: []*v4.Repository{
				{
					Id:   "0",
					Name: "cpe:/o:redhat:enterprise_linux:9::fastdatapath",
					Key:  "rhel-cpe-repository",
					Cpe:  "cpe:2.3:o:redhat:enterprise_linux:9:*:fastdatapath:*:*:*:*:*",
				},
				{
					Id:   "1",
					Name: "cpe:/a:redhat:openshift:4.16::el9",
					Key:  "rhel-cpe-repository",
					Cpe:  "cpe:2.3:a:redhat:openshift:4.16:*:el9:*:*:*:*:*",
				},
			},
			Environments: map[string]*v4.Environment_List{
				"1": {
					Environments: []*v4.Environment{
						{
							PackageDb:     "sqlite:usr/share/rpm",
							IntroducedIn:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
							RepositoryIds: []string{"0", "1"},
						},
					},
				},
			},
		},
		Notes: []v4.VulnerabilityReport_Note{v4.VulnerabilityReport_NOTE_OS_UNKNOWN},
	}

	mockOutOfBoundsVulnReport = &v4.VulnerabilityReport{
		HashId: "",
		Vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
			"7401229": {
				Id:                 "7401229",
				Name:               "RHSA-2024:4616",
				Description:        "Sample Description",
				Severity:           "Moderate",
				NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_MODERATE,
				FixedInVersion:     "0:4.16.0-202407111006.p0.gfa84651.assembly.stream.el9",
				Cvss: &v4.VulnerabilityReport_Vulnerability_CVSS{
					V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
						BaseScore: 7.5,
						Vector:    "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H",
					},
				},
			},
		},
		PackageVulnerabilities: map[string]*v4.StringList{
			"0": {
				// 0000000 is an unknown Vulnerability ID and should be skipped
				Values: []string{"0000000", "7401229"},
			},
			"1": {
				Values: []string{"7401229"},
			},
		},
		Contents: &v4.Contents{
			Packages: []*v4.Package{
				{
					Id:      "0",
					Name:    "openssh-clients",
					Version: "8.7p1-38.el9",
					Kind:    "binary",
					Source: &v4.Package{
						Name:    "openssh",
						Version: "8.7p1-38.el9",
						Kind:    "source",
						Source:  nil,
						Cpe:     "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
					},
					PackageDb:      "sqlite:usr/share/rpm",
					RepositoryHint: "hash:sha256:f52ca767328e6919ec11a1da654e92743587bd3c008f0731f8c4de3af19c1830|key:199e2f91fd431d51",
					Arch:           "x86_64",
					Cpe:            "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
				},
				{
					Id:      "1",
					Name:    "skopeo",
					Version: "2:1.14.4-2.rhaos4.16.el9",
					Kind:    "binary",
					Source: &v4.Package{
						Name:    "skopeo",
						Version: "2:1.14.4-2.rhaos4.16.el9",
						Kind:    "source",
						Cpe:     "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
					},
					PackageDb:      "sqlite:usr/share/rpm",
					RepositoryHint: "hash:sha256:072a75d1b9b36457751ef05031fd69615f21ebaa935c30d74d827328b78fa694|key:199e2f91fd431d51",
					Arch:           "x86_64",
					Cpe:            "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
				},
			},
			Repositories: []*v4.Repository{
				{
					Id:   "0",
					Name: "cpe:/o:redhat:enterprise_linux:9::fastdatapath",
					Key:  "rhel-cpe-repository",
					Cpe:  "cpe:2.3:o:redhat:enterprise_linux:9:*:fastdatapath:*:*:*:*:*",
				},
				{
					Id:   "1",
					Name: "cpe:/a:redhat:openshift:4.16::el9",
					Key:  "rhel-cpe-repository",
					Cpe:  "cpe:2.3:a:redhat:openshift:4.16:*:el9:*:*:*:*:*",
				},
			},
			Environments: map[string]*v4.Environment_List{
				"1": {
					Environments: []*v4.Environment{
						{
							PackageDb:     "sqlite:usr/share/rpm",
							IntroducedIn:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
							RepositoryIds: []string{"0", "1"},
						},
					},
				},
			},
		},
		Notes: []v4.VulnerabilityReport_Note{v4.VulnerabilityReport_NOTE_OS_UNKNOWN},
	}
)

func TestNodeScan(t *testing.T) {
	expected := &storage.EmbeddedNodeScanComponent{
		Name:    "openssh-clients",
		Version: "8.7p1-38.el9",
		Vulns: []*storage.EmbeddedVulnerability{
			{
				Cve:          "RHSA-2024:4616",
				Cvss:         7.5,
				Summary:      "Sample Description",
				Link:         "https://localhost/7401229",
				ScoreVersion: storage.EmbeddedVulnerability_V3,
				SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
					FixedBy: "0:4.16.0-202407111006.p0.gfa84651.assembly.stream.el9",
				},
				VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
				Severity:          storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
				CvssV3: &storage.CVSSV3{
					Vector:              "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H",
					ExploitabilityScore: 3.9,
					ImpactScore:         3.6,
					AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
					AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
					PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_NONE,
					UserInteraction:     storage.CVSSV3_UI_NONE,
					Scope:               storage.CVSSV3_UNCHANGED,
					Confidentiality:     storage.CVSSV3_IMPACT_NONE,
					Integrity:           storage.CVSSV3_IMPACT_NONE,
					Availability:        storage.CVSSV3_IMPACT_HIGH,
					Score:               7.5,
					Severity:            storage.CVSSV3_HIGH,
				},
			},
		},
	}

	actual := nodeScan("Red Hat Enterprise Linux CoreOS 417.94.202409121747-0", mockVulnReport)

	assert.Equal(t, storage.NodeScan_SCANNER_V4, actual.GetScannerVersion())
	assert.Len(t, actual.GetComponents(), 2)
	assert.Empty(t, actual.GetNotes())
	protoassert.SliceContains(t, actual.GetComponents(), expected)
}

func TestNodeScan_EmptyVulnerabilityReport(t *testing.T) {
	r := &v4.VulnerabilityReport{
		HashId:                 "",
		Vulnerabilities:        nil,
		PackageVulnerabilities: nil,
		Contents:               nil,
		Notes:                  nil,
	}

	var actual *storage.NodeScan
	assert.NotPanics(t, func() {
		actual = nodeScan("Red Hat Enterprise Linux CoreOS 417.94.202409121747-0", r)
	})
	assert.NotNil(t, actual)
	assert.Equal(t, storage.NodeScan_SCANNER_V4, actual.GetScannerVersion())

}

func TestNodeComponents_OutOfBounds(t *testing.T) {
	expectedCVE := &storage.EmbeddedVulnerability{
		Cve:               "RHSA-2024:4616",
		Cvss:              8.2,
		Summary:           "Sample Description",
		ScoreVersion:      storage.EmbeddedVulnerability_V3,
		SetFixedBy:        &storage.EmbeddedVulnerability_FixedBy{FixedBy: "0:4.16.0-202407111006.p0.gfa84651.assembly.stream.el9"},
		VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
		Severity:          storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
		Link:              "https://access.redhat.com/errata/RHSA-2024:4616",
		CvssV3: &storage.CVSSV3{
			Vector:              "CVSS:3.1/AV:A/AC:L/PR:N/UI:N/S:C/C:L/I:N/A:H",
			ExploitabilityScore: 2.8,
			ImpactScore:         4.7,
			AttackVector:        storage.CVSSV3_ATTACK_ADJACENT,
			AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
			PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_NONE,
			UserInteraction:     storage.CVSSV3_UI_NONE,
			Scope:               storage.CVSSV3_CHANGED,
			Confidentiality:     storage.CVSSV3_IMPACT_LOW,
			Integrity:           storage.CVSSV3_IMPACT_NONE,
			Availability:        storage.CVSSV3_IMPACT_HIGH,
			Score:               8.2,
			Severity:            storage.CVSSV3_HIGH,
		},
		CvssMetrics: []*storage.CVSSScore{
			{
				CvssScore: &storage.CVSSScore_Cvssv3{
					Cvssv3: &storage.CVSSV3{
						Vector:              "CVSS:3.1/AV:A/AC:L/PR:N/UI:N/S:C/C:L/I:N/A:H",
						ExploitabilityScore: 2.8,
						ImpactScore:         4.7,
						AttackVector:        storage.CVSSV3_ATTACK_ADJACENT,
						AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
						PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_NONE,
						UserInteraction:     storage.CVSSV3_UI_NONE,
						Scope:               storage.CVSSV3_CHANGED,
						Confidentiality:     storage.CVSSV3_IMPACT_LOW,
						Integrity:           storage.CVSSV3_IMPACT_NONE,
						Availability:        storage.CVSSV3_IMPACT_HIGH,
						Score:               8.2,
						Severity:            storage.CVSSV3_HIGH,
					},
				},
				Source: storage.Source_SOURCE_RED_HAT,
				Url:    "https://access.redhat.com/errata/RHSA-2024:4616",
			},
			{
				CvssScore: &storage.CVSSScore_Cvssv2{
					Cvssv2: &storage.CVSSV2{
						Vector:              "AV:N/AC:M/Au:M/C:C/I:N/A:P",
						AttackVector:        storage.CVSSV2_ATTACK_NETWORK,
						AccessComplexity:    storage.CVSSV2_ACCESS_MEDIUM,
						Authentication:      storage.CVSSV2_AUTH_MULTIPLE,
						Confidentiality:     storage.CVSSV2_IMPACT_COMPLETE,
						Integrity:           storage.CVSSV2_IMPACT_NONE,
						Availability:        storage.CVSSV2_IMPACT_PARTIAL,
						ExploitabilityScore: 5.5,
						ImpactScore:         7.8,
						Score:               6.4,
						Severity:            storage.CVSSV2_MEDIUM,
					},
				},
				Source: storage.Source_SOURCE_NVD,
				Url:    "https://nvd.nist.gov/vuln/detail/CVE-1234-567",
			},
		},
	}

	actual := nodeComponents(mockOutOfBoundsVulnReport)
	assert.Len(t, actual, 2)
	for _, c := range actual {
		// Ensure that each of the components track the expected CVE
		protoassert.SliceContains(t, c.GetVulns(), expectedCVE)
	}
}

func TestNodeComponents_NoVulns(t *testing.T) {
	r := &v4.VulnerabilityReport{
		Contents: &v4.Contents{
			Packages: []*v4.Package{
				{
					Id: "DOESNOTEXIST",
				},
			},
		},
		PackageVulnerabilities: map[string]*v4.StringList{
			"1": {
				Values: []string{"CVE1-ID"},
			},
		},
	}
	got := nodeComponents(r)
	assert.Len(t, got, 1)
	assert.Empty(t, got[0])
}

func TestNodeComponents_MissingVuln(t *testing.T) {
	r := &v4.VulnerabilityReport{
		PackageVulnerabilities: map[string]*v4.StringList{
			"1": {
				Values: []string{"DOESNTEXIST", "V2"},
			},
		},
		Vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
			"V2": {
				Id:                 "V2",
				Name:               "CVE-Name",
				FixedInVersion:     "v99",
				NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_IMPORTANT,
			},
		},
	}
	got := nodeComponents(r)
	assert.Len(t, got, 1)
	assert.Len(t, got[0].GetVulnerabilities(), 1)
	assert.Equal(t, "CVE-Name", got[0].GetVulnerabilities()[0].GetCveBaseInfo().GetCve())
}

func TestNodeOS(t *testing.T) {
	testCases := map[string]struct {
		osImage  string
		expected string
	}{
		"realistic version": {
			osImage:  "Red Hat Enterprise Linux CoreOS 417.94.202409121747-0",
			expected: "rhcos:4.17",
		},
		"realistic long version": {
			osImage:  "Red Hat Enterprise Linux CoreOS 41712345.94.2024",
			expected: "rhcos:4.1712345",
		},
		"non-RHCOS": {
			osImage:  "Oracle Linux Server release 6.8",
			expected: "",
		},
		"blank": {
			osImage:  "",
			expected: "",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			os := nodeOS(tc.osImage)
			assert.Equal(t, tc.expected, os)
		})
	}
}

func TestNodeNotes(t *testing.T) {
	testCases := map[string]struct {
		report   *v4.VulnerabilityReport
		osImage  string
		expected []storage.NodeScan_Note
	}{
		"basic unspecified": {
			report: &v4.VulnerabilityReport{
				Notes: []v4.VulnerabilityReport_Note{v4.VulnerabilityReport_NOTE_UNSPECIFIED},
			},
			expected: []storage.NodeScan_Note{storage.NodeScan_UNSET},
		},
		"basic os unknown": {
			report: &v4.VulnerabilityReport{
				Notes: []v4.VulnerabilityReport_Note{v4.VulnerabilityReport_NOTE_OS_UNKNOWN},
			},
			expected: []storage.NodeScan_Note{storage.NodeScan_UNSUPPORTED},
		},
		"basic os unsupported": {
			report: &v4.VulnerabilityReport{
				Notes: []v4.VulnerabilityReport_Note{v4.VulnerabilityReport_NOTE_OS_UNSUPPORTED},
			},
			expected: []storage.NodeScan_Note{storage.NodeScan_UNSUPPORTED},
		},
		"RHCOS remove unsupported note": {
			report: &v4.VulnerabilityReport{
				Notes: []v4.VulnerabilityReport_Note{
					v4.VulnerabilityReport_NOTE_OS_UNSUPPORTED,
					v4.VulnerabilityReport_NOTE_UNSPECIFIED,
				},
			},
			osImage:  "Red Hat Enterprise Linux CoreOS 417.94.202409121747-0",
			expected: []storage.NodeScan_Note{storage.NodeScan_UNSET},
		},
		"non-RHCOS keep all notes": {
			report: &v4.VulnerabilityReport{
				Notes: []v4.VulnerabilityReport_Note{
					v4.VulnerabilityReport_NOTE_OS_UNSUPPORTED,
					v4.VulnerabilityReport_NOTE_UNSPECIFIED,
				},
			},
			osImage:  "Red Hat Enterprise Linux CoreOS 417.94.202409121747-0",
			expected: []storage.NodeScan_Note{storage.NodeScan_UNSUPPORTED, storage.NodeScan_UNSET},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got := nodeNotes(tc.report, "")
			assert.Len(t, got, 1)
			assert.ElementsMatch(t, tc.expected, got)
		})
	}
}
