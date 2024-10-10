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

	actual := ToNodeScan(r)

	s.Equal(storage.NodeScan_SCANNER_V4, actual.GetScannerVersion())
	s.Len(actual.GetComponents(), 1)
	s.Equal("openssh-clients", actual.GetComponents()[0].GetName())
	s.Equal("8.7p1-38.el9", actual.GetComponents()[0].GetVersion())
	s.Equal("RHSA-2024:4616", actual.GetComponents()[0].GetVulns()[0].GetCve())
	s.Equal([]storage.NodeScan_Note{storage.NodeScan_UNSUPPORTED}, actual.GetNotes())
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
		actual = ToNodeScan(r)
	})

	s.NotNil(actual)
	s.Equal(storage.NodeScan_SCANNER_V4, actual.GetScannerVersion())

}

func (s *indexReportConvertSuite) TestToStorageComponentsOutOfBounds() {
	in := createOutOfBoundsReport()
	expectedCVE := &storage.EmbeddedVulnerability{
		Cve:               "RHSA-2024:4616",
		Summary:           "Sample Description",
		SetFixedBy:        &storage.EmbeddedVulnerability_FixedBy{FixedBy: "0:4.16.0-202407111006.p0.gfa84651.assembly.stream.el9"},
		VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
		Severity:          storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
	}

	actual := toStorageComponents(in)

	s.Len(actual, 2)
	for _, c := range actual {
		// Ensure that each of the components track the expected CVE
		protoassert.SliceContains(s.T(), c.GetVulns(), expectedCVE)
	}
}

func (s *indexReportConvertSuite) TestConvertNodeNotes() {
	in := []v4.VulnerabilityReport_Note{v4.VulnerabilityReport_NOTE_UNSPECIFIED, v4.VulnerabilityReport_NOTE_OS_UNKNOWN, v4.VulnerabilityReport_NOTE_OS_UNSUPPORTED}
	expected := []storage.NodeScan_Note{storage.NodeScan_UNSET, storage.NodeScan_UNSUPPORTED, storage.NodeScan_UNSUPPORTED}

	actual := toStorageNotes(in)
	for i, note := range actual {
		s.Equal(note, expected[i])
	}
}

func (s *indexReportConvertSuite) TestConvertVulnerability() {
	v := &v4.VulnerabilityReport_Vulnerability{
		Name:           "TestCVE",
		Description:    "Test Description",
		Link:           "https://some.localhost",
		Severity:       "Low",
		FixedInVersion: "2.4.54-r3",
		CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
			{
				Url: "https://dontpickme.localhost",
			},
			{
				V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
					BaseScore: 1,
				},
				Url: "https://dontpickme.either.localhost",
			},
			{
				V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
					BaseScore: 7,
					Vector:    "CVSS:3.0/AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:L/A:H",
				},
				Url: "https://url1.localhost",
			},
			{
				V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
					BaseScore: 5,
					Vector:    "CVSS:3.0/AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:L/A:H",
				},
				Url: "https://url2.localhost",
			},
		},
	}
	expected := &storage.EmbeddedVulnerability{
		Cve:               "TestCVE",
		Summary:           "Test Description",
		SetFixedBy:        &storage.EmbeddedVulnerability_FixedBy{FixedBy: "2.4.54-r3"},
		VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
		Severity:          storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		Link:              "https://url1.localhost",
		Cvss:              5,
		ScoreVersion:      1,
		CvssV3: &storage.CVSSV3{
			Vector:             "CVSS:3.0/AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:L/A:H",
			Confidentiality:    storage.CVSSV3_IMPACT_NONE,
			Integrity:          storage.CVSSV3_IMPACT_LOW,
			Availability:       storage.CVSSV3_IMPACT_HIGH,
			PrivilegesRequired: storage.CVSSV3_PRIVILEGE_LOW,
			Severity:           storage.CVSSV3_MEDIUM,
			Score:              5,
		},
	}

	actual := convertVulnerability(v)

	protoassert.Equal(s.T(), expected, actual)
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
				Cvss: &v4.VulnerabilityReport_Vulnerability_CVSS{
					V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
						BaseScore: 7.5,
						Vector:    "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H",
					},
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
