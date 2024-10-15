package scannerv4

import (
	"testing"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/suite"
)

func TestIndexReportConvert(t *testing.T) {
	suite.Run(t, new(indexReportConvertSuite))
}

type indexReportConvertSuite struct {
	suite.Suite
}

func (s *indexReportConvertSuite) TestToNodeInventory() {
	r := createVulnerabilityReport()
	expected := &storage.EmbeddedNodeScanComponent{
		Name:    "openssh-clients",
		Version: "8.7p1-38.el9",
		Vulns: []*storage.EmbeddedVulnerability{{
			Cve:               "RHSA-2024:4616",
			Cvss:              7.5,
			Summary:           "Sample Description",
			Link:              "https://localhost/7401229",
			ScoreVersion:      1,
			SetFixedBy:        &storage.EmbeddedVulnerability_FixedBy{FixedBy: "0:4.16.0-202407111006.p0.gfa84651.assembly.stream.el9"},
			VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
			Severity:          storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
			CvssV3: &storage.CVSSV3{
				Vector:              "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H",
				ExploitabilityScore: 3.9,
				ImpactScore:         3.6,
				AttackVector:        2,
				Availability:        2,
				Score:               7.5,
				Severity:            storage.CVSSV3_HIGH,
			},
		}},
	}

	actual := toNodeScan(r, "Red Hat Enterprise Linux CoreOS 417.94.202409121747-0")

	s.Equal(storage.NodeScan_SCANNER_V4, actual.GetScannerVersion())
	s.Len(actual.GetComponents(), 2)
	s.Len(actual.GetNotes(), 0)
	protoassert.SliceContains(s.T(), actual.GetComponents(), expected)
}

func (s *indexReportConvertSuite) TestEmptyReportConversionNoPanic() {
	r := &v4.VulnerabilityReport{
		HashId:                 "",
		Vulnerabilities:        nil,
		PackageVulnerabilities: nil,
		Contents:               nil,
		Notes:                  nil,
	}

	var actual *storage.NodeScan

	s.NotPanics(func() {
		actual = toNodeScan(r, "Red Hat Enterprise Linux CoreOS 417.94.202409121747-0")
	})

	s.NotNil(actual)
	s.Equal(storage.NodeScan_SCANNER_V4, actual.GetScannerVersion())

}

func (s *indexReportConvertSuite) TestToStorageComponentsOutOfBounds() {
	in := createOutOfBoundsReport()
	expectedCVE := &storage.EmbeddedVulnerability{
		Cve:               "RHSA-2024:4616",
		Cvss:              7.5,
		Summary:           "Sample Description",
		ScoreVersion:      1,
		SetFixedBy:        &storage.EmbeddedVulnerability_FixedBy{FixedBy: "0:4.16.0-202407111006.p0.gfa84651.assembly.stream.el9"},
		VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
		Severity:          storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
		CvssV3: &storage.CVSSV3{
			Vector:              "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H",
			ExploitabilityScore: 3.9,
			ImpactScore:         3.6,
			AttackVector:        2,
			Availability:        2,
			Score:               7.5,
			Severity:            4,
		},
	}

	actual := toStorageComponents(in)

	s.Len(actual, 2)
	for _, c := range actual {
		// Ensure that each of the components track the expected CVE
		// protoassert.SliceContains(s.T(), c.GetVulns(), expectedCVE)
		protoassert.Equal(s.T(), expectedCVE, c.GetVulns()[0])
	}
}

func (s *indexReportConvertSuite) TestGetPackageVulnsBrokenMapping() {
	r := &v4.VulnerabilityReport{
		PackageVulnerabilities: map[string]*v4.StringList{"1": {Values: []string{"CVE1-ID"}}},
	}
	actual := getPackageVulns("DOESNTEXIST", r)
	s.Len(actual, 0)
}

func (s *indexReportConvertSuite) TestGetPackageVulnsBrokenVulnereability() {
	r := &v4.VulnerabilityReport{
		PackageVulnerabilities: map[string]*v4.StringList{"1": {Values: []string{"DOESNTEXIST", "V2"}}},
		Vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
			"V2": {
				Id:                 "V2",
				Name:               "CVE-Name",
				FixedInVersion:     "v99",
				NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_IMPORTANT,
			},
		},
	}
	actual := getPackageVulns("1", r)
	s.Len(actual, 1)
	s.Equal("CVE-Name", actual[0].GetCve())
}

func (s *indexReportConvertSuite) TestConvertNodeNotes() {
	in := []v4.VulnerabilityReport_Note{v4.VulnerabilityReport_NOTE_UNSPECIFIED, v4.VulnerabilityReport_NOTE_OS_UNKNOWN, v4.VulnerabilityReport_NOTE_OS_UNSUPPORTED}
	expected := []storage.NodeScan_Note{storage.NodeScan_UNSET, storage.NodeScan_UNSUPPORTED, storage.NodeScan_UNSUPPORTED}

	actual := toStorageNotes(in)
	for i, note := range actual {
		s.Equal(note, expected[i])
	}
}

func (s *indexReportConvertSuite) TestToOperatingSystem() {
	cases := map[string]struct {
		in       string
		expected string
	}{
		"realistic version": {
			in:       "Red Hat Enterprise Linux CoreOS 417.94.202409121747-0",
			expected: "rhcos:4.17",
		},
		"realistic long version": {
			in:       "Red Hat Enterprise Linux CoreOS 41712345.94.2024",
			expected: "rhcos:4.1712345",
		},
		"non-RHCOS": {
			in:       "Oracle Linux Server release 6.8",
			expected: "",
		},
		"blank": {
			in:       "",
			expected: "",
		},
	}
	for name, c := range cases {
		s.T().Run(name, func(tt *testing.T) {
			actual := toOperatingSystem(c.in)
			s.Equal(c.expected, actual)
		})
	}
}

func (s *indexReportConvertSuite) TestFixNotes() {
	cases := map[string]struct {
		in       []storage.NodeScan_Note
		osRef    string
		expected []storage.NodeScan_Note
	}{
		"RHCOS": {
			in:       []storage.NodeScan_Note{storage.NodeScan_UNSUPPORTED, storage.NodeScan_UNSET},
			osRef:    "Red Hat Enterprise Linux CoreOS 417.94.202409121747-0",
			expected: []storage.NodeScan_Note{storage.NodeScan_UNSET},
		},
		"Non-RHCOS": {
			in:       []storage.NodeScan_Note{storage.NodeScan_UNSUPPORTED, storage.NodeScan_UNSET},
			osRef:    "Oracle Linux Server release 6.8",
			expected: []storage.NodeScan_Note{storage.NodeScan_UNSUPPORTED, storage.NodeScan_UNSET},
		},
	}
	for name, c := range cases {
		s.T().Run(name, func(tt *testing.T) {
			actual := fixNotes(c.in, c.osRef)
			s.Equal(c.expected, actual)
		})
	}
}

func createVulnerabilityReport() *v4.VulnerabilityReport {
	return &v4.VulnerabilityReport{
		HashId: "",
		Vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
			"7401229": {
				Id:                 "7401229",
				Name:               "RHSA-2024:4616",
				Description:        "Sample Description",
				Severity:           "Moderate",
				NormalizedSeverity: 2,
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
			"0": {Values: []string{"7401229"}},
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
			Environments: map[string]*v4.Environment_List{"1": {Environments: []*v4.Environment{
				{
					PackageDb:     "sqlite:usr/share/rpm",
					IntroducedIn:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					RepositoryIds: []string{"0", "1"},
				},
			},
			}},
		},
		Notes: []v4.VulnerabilityReport_Note{v4.VulnerabilityReport_NOTE_OS_UNKNOWN},
	}
}

func createOutOfBoundsReport() *v4.VulnerabilityReport {
	return &v4.VulnerabilityReport{
		HashId: "",
		Vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
			"7401229": {
				Id:                 "7401229",
				Name:               "RHSA-2024:4616",
				Description:        "Sample Description",
				Severity:           "Moderate",
				NormalizedSeverity: 2,
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
			"0": {Values: []string{"0000000", "7401229"}}, // 0000000 is an unknown Vulnerability ID and should be skipped
			"1": {Values: []string{"7401229"}},
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
			Environments: map[string]*v4.Environment_List{"1": {Environments: []*v4.Environment{
				{
					PackageDb:     "sqlite:usr/share/rpm",
					IntroducedIn:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					RepositoryIds: []string{"0", "1"},
				},
			},
			}},
		},
		Notes: []v4.VulnerabilityReport_Note{v4.VulnerabilityReport_NOTE_OS_UNKNOWN},
	}
}
