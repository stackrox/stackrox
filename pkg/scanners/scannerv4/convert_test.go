package scannerv4

import (
	"fmt"
	"testing"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
)

func TestNoPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		imageScan(nil, nil)

		report := &v4.VulnerabilityReport{}
		imageScan(nil, report)

		report.Contents = &v4.Contents{}
		imageScan(nil, report)

		report.Contents.Packages = []*v4.Package{}
		imageScan(nil, report)

		report.Contents.Packages = append(report.Contents.Packages, &v4.Package{
			Id: "1",
		})
		imageScan(nil, report)

		report.PackageVulnerabilities = map[string]*v4.StringList{}
		imageScan(nil, report)

		report.PackageVulnerabilities["1"] = &v4.StringList{}
		imageScan(nil, report)

		report.PackageVulnerabilities["1"].Values = []string{}
		imageScan(nil, report)

		report.PackageVulnerabilities["1"].Values = []string{"CVE1"}
		imageScan(nil, report)
	})
}

func TestConvert(t *testing.T) {
	inMetadata := &storage.ImageMetadata{
		V2: &storage.V2Metadata{},
		V1: &storage.V1Metadata{
			Layers: []*storage.ImageLayer{
				{Empty: false},
			},
		},
		LayerShas: []string{"hash"},
	}

	inReport := &v4.VulnerabilityReport{
		Contents: &v4.Contents{
			Environments: map[string]*v4.Environment_List{
				"1": {
					Environments: []*v4.Environment{
						{
							PackageDb:    "maven:opt/java/pkg.jar",
							IntroducedIn: "hash",
						},
					},
				},
			},
			Distributions: []*v4.Distribution{
				{
					Did:       "rhel",
					VersionId: "9",
				},
			},
			Packages: []*v4.Package{
				{
					Id:      "1",
					Name:    "my-java-pkg",
					Version: "1.2.3",
				},
			},
		},
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
	}

	expected := &storage.ImageScan{
		Components: []*storage.EmbeddedImageScanComponent{
			{
				Name:          "my-java-pkg",
				Version:       "1.2.3",
				Source:        storage.SourceType_JAVA,
				Location:      "opt/java/pkg.jar",
				HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
				Vulns: []*storage.EmbeddedVulnerability{
					{
						Cve:               "CVE1-Name",
						VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
						Severity:          storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
						SetFixedBy:        &storage.EmbeddedVulnerability_FixedBy{FixedBy: "v99"},
					},
				},
			},
		},
		OperatingSystem: "rhel:9",
	}

	actual := imageScan(inMetadata, inReport)

	protoassert.SlicesEqual(t, expected.Components, actual.Components)
	assert.Equal(t, expected.OperatingSystem, actual.OperatingSystem)
}

func TestComponents(t *testing.T) {
	testcases := []struct {
		name     string
		metadata *storage.ImageMetadata
		report   *v4.VulnerabilityReport
		expected []*storage.EmbeddedImageScanComponent
	}{
		{
			name: "basic no vulns",
			metadata: &storage.ImageMetadata{
				V1: &storage.V1Metadata{
					Digest: "some V1 digest",
					Layers: []*storage.ImageLayer{
						{
							Empty: true,
						},
						{
							Empty: true,
						},
						{
							Empty: true,
						},
						{
							Instruction: "RUN",
							Value:       "mv -fZ /tmp/ubi.repo /etc/yum.repos.d/ubi.repo || :",
						},
						{
							Instruction: "COPY",
							Value:       "--chmod=755 ./sleepforever.sh /sleepforever.sh # buildkit",
						},
						{
							Empty: true,
						},
					},
				},
				V2: &storage.V2Metadata{
					Digest: "some V2 digest",
				},
				LayerShas: []string{"layer1", "layer2"},
				DataSource: &storage.DataSource{
					Id:   "dataSourceID",
					Name: "dataSourceName",
				},
			},
			report: &v4.VulnerabilityReport{
				HashId: "hashID",
				Contents: &v4.Contents{
					Packages: []*v4.Package{
						{
							Id:      "1",
							Name:    "glib2",
							Version: "2.68.4-14.el9",
						},
					},
					Distributions: []*v4.Distribution{
						{
							Id:        "1",
							Did:       "rhel",
							VersionId: "9",
						},
					},
					Environments: map[string]*v4.Environment_List{
						"1": {
							Environments: []*v4.Environment{
								{
									PackageDb:      "sqlite:var/lib/rpm",
									IntroducedIn:   "layer1",
									DistributionId: "1",
									RepositoryIds:  []string{"0"},
								},
							},
						},
					},
				},
			},
			expected: []*storage.EmbeddedImageScanComponent{
				{
					Name:     "glib2",
					Version:  "2.68.4-14.el9",
					Source:   storage.SourceType_OS,
					Location: "var/lib/rpm",
					HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{
						LayerIndex: 3,
					},
				},
			},
		},
		{
			name: "layer mismatch, no layer indexes",
			metadata: &storage.ImageMetadata{
				V1: &storage.V1Metadata{
					Digest: "some V1 digest",
					Layers: []*storage.ImageLayer{
						{
							Empty: true,
						},
						{
							Empty: true,
						},
						{
							Empty: true,
						},
						{
							Instruction: "RUN",
							Value:       "mv -fZ /tmp/ubi.repo /etc/yum.repos.d/ubi.repo || :",
						},
						{
							Instruction: "COPY",
							Value:       "--chmod=755 ./sleepforever.sh /sleepforever.sh # buildkit",
						},
						{
							Empty: true,
						},
					},
				},
				V2: &storage.V2Metadata{
					Digest: "some V2 digest",
				},
				LayerShas: []string{"layer1", "layer2"},
				DataSource: &storage.DataSource{
					Id:   "dataSourceID",
					Name: "dataSourceName",
				},
			},
			report: &v4.VulnerabilityReport{
				HashId: "hashID",
				Contents: &v4.Contents{
					Packages: []*v4.Package{
						{
							Id:      "1",
							Name:    "glib2",
							Version: "2.68.4-14.el9",
						},
					},
					Distributions: []*v4.Distribution{
						{
							Id:        "1",
							Did:       "rhel",
							VersionId: "9",
						},
					},
					Environments: map[string]*v4.Environment_List{
						"1": {
							Environments: []*v4.Environment{
								{
									PackageDb:      "sqlite:var/lib/rpm",
									IntroducedIn:   "some layer which does not exist in the image",
									DistributionId: "1",
									RepositoryIds:  []string{"0"},
								},
							},
						},
					},
				},
			},
			expected: []*storage.EmbeddedImageScanComponent{
				{
					Name:          "glib2",
					Version:       "2.68.4-14.el9",
					Source:        storage.SourceType_OS,
					Location:      "var/lib/rpm",
					HasLayerIndex: nil,
				},
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			got := components(tc.metadata, tc.report)
			protoassert.SlicesEqual(t, tc.expected, got, fmt.Sprintf("expected: %+#v\ngot: %+#v", tc.expected, got))
		})
	}
}

func TestSetEPSS(t *testing.T) {
	testcases := []struct {
		name       string
		epssDetail *v4.VulnerabilityReport_Vulnerability_EPSS
		expected   *storage.EPSS
	}{
		{
			name: "EPSS OK",
			epssDetail: &v4.VulnerabilityReport_Vulnerability_EPSS{
				Date:         "test-date",
				ModelVersion: "test-model-version",
				Probability:  0.91963,
				Percentile:   0.99030,
			},
			expected: &storage.EPSS{EpssProbability: 0.91963, EpssPercentile: 0.99030},
		},
		{
			name: "EPSS really small",
			epssDetail: &v4.VulnerabilityReport_Vulnerability_EPSS{
				Date:         "test-date",
				ModelVersion: "test-model-version",
				Probability:  0.00042,
				Percentile:   0.003,
			},
			expected: &storage.EPSS{EpssProbability: 0.00042, EpssPercentile: 0.003},
		},
		{
			name: "EPSS 0 probability",
			epssDetail: &v4.VulnerabilityReport_Vulnerability_EPSS{
				Date:         "test-date",
				ModelVersion: "test-model-version",
				Probability:  0,
				Percentile:   0.003,
			},
			expected: &storage.EPSS{EpssProbability: 0, EpssPercentile: 0.003},
		},
		{
			name: "EPSS 0 percentile",
			epssDetail: &v4.VulnerabilityReport_Vulnerability_EPSS{
				Date:         "test-date",
				ModelVersion: "test-model-version",
				Probability:  0.000426,
				Percentile:   0,
			},
			expected: &storage.EPSS{EpssProbability: 0.000426, EpssPercentile: 0},
		},
		{
			name:       "EPSS nil input",
			epssDetail: nil,
			expected:   nil,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			result := epss(testcase.epssDetail)
			if testcase.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}
			if result == nil {
				t.Errorf("expected %+v, got nil", testcase.expected)
				return
			}
			if result.EpssProbability != testcase.expected.EpssProbability || result.EpssPercentile != testcase.expected.EpssPercentile {
				t.Errorf("expected %+v, got %+v", testcase.expected, result)
			}
		})
	}
}

func TestSetScoresAndScoreVersions(t *testing.T) {
	testcases := []struct {
		name        string
		cvssMetrics []*v4.VulnerabilityReport_Vulnerability_CVSS
		expected    *storage.EmbeddedVulnerability
		wantErr     bool
	}{
		{
			name: "CVSS 3.1 and CVSS 2 from one data source",
			cvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
				{
					V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
						BaseScore: 8.2,
						Vector:    "CVSS:3.1/AV:A/AC:L/PR:N/UI:N/S:C/C:L/I:N/A:H",
					},
					V2: &v4.VulnerabilityReport_Vulnerability_CVSS_V2{
						BaseScore: 6.4,
						Vector:    "AV:N/AC:M/Au:M/C:C/I:N/A:P",
					},
					Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
					Url:    "https://nvd.nist.gov/vuln/detail/CVE-1234-567",
				},
			},
			expected: &storage.EmbeddedVulnerability{
				Cvss:         8.2,
				ScoreVersion: storage.EmbeddedVulnerability_V3,
				Link:         "https://nvd.nist.gov/vuln/detail/CVE-1234-567",
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
				CvssV2: &storage.CVSSV2{
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
						Source: storage.Source_SOURCE_NVD,
						Url:    "https://nvd.nist.gov/vuln/detail/CVE-1234-567",
					},
				},
			},
		},
		{
			name: "CVSS 2 score differs from calculated",
			cvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
				{
					V2: &v4.VulnerabilityReport_Vulnerability_CVSS_V2{
						BaseScore: 1.2,
						Vector:    "AV:N/AC:M/Au:M/C:C/I:N/A:P",
					},
					Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
					Url:    "https://nvd.nist.gov/vuln/detail/CVE-1234-567",
				},
			},
			expected: &storage.EmbeddedVulnerability{
				Link:         "https://nvd.nist.gov/vuln/detail/CVE-1234-567",
				Cvss:         1.2,
				ScoreVersion: storage.EmbeddedVulnerability_V2,
				CvssV2: &storage.CVSSV2{
					Vector:              "AV:N/AC:M/Au:M/C:C/I:N/A:P",
					AttackVector:        storage.CVSSV2_ATTACK_NETWORK,
					AccessComplexity:    storage.CVSSV2_ACCESS_MEDIUM,
					Authentication:      storage.CVSSV2_AUTH_MULTIPLE,
					Confidentiality:     storage.CVSSV2_IMPACT_COMPLETE,
					Integrity:           storage.CVSSV2_IMPACT_NONE,
					Availability:        storage.CVSSV2_IMPACT_PARTIAL,
					ExploitabilityScore: 5.5,
					ImpactScore:         7.8,
					Score:               1.2,
					Severity:            storage.CVSSV2_LOW,
				},
				CvssMetrics: []*storage.CVSSScore{
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
								Score:               1.2,
								Severity:            storage.CVSSV2_LOW,
							},
						},
						Source: storage.Source_SOURCE_NVD,
						Url:    "https://nvd.nist.gov/vuln/detail/CVE-1234-567",
					},
				},
			},
		},
		{
			name: "CVSS 3.1 score differs from calculated",
			cvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
				{
					V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
						BaseScore: 4.0,
						Vector:    "CVSS:3.1/AV:A/AC:L/PR:N/UI:N/S:C/C:L/I:N/A:H",
					},
					Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
					Url:    "https://nvd.nist.gov/vuln/detail/CVE-1234-567",
				},
			},
			expected: &storage.EmbeddedVulnerability{
				Cvss:         4.0,
				ScoreVersion: storage.EmbeddedVulnerability_V3,
				Link:         "https://nvd.nist.gov/vuln/detail/CVE-1234-567",
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
					Score:               4.0,
					Severity:            storage.CVSSV3_MEDIUM,
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
								Score:               4.0,
								Severity:            storage.CVSSV3_MEDIUM,
							},
						},
						Source: storage.Source_SOURCE_NVD,
						Url:    "https://nvd.nist.gov/vuln/detail/CVE-1234-567",
					},
				},
			},
		},
		{
			name: "Both CVSS 3.1 and CVSS 2",
			cvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
				{
					V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
						BaseScore: 8.2,
						Vector:    "CVSS:3.1/AV:A/AC:L/PR:N/UI:N/S:C/C:L/I:N/A:H",
					},
					Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_RED_HAT,
					Url:    "https://access.redhat.com/security/cve/CVE-1234-567",
				},
				{
					V2: &v4.VulnerabilityReport_Vulnerability_CVSS_V2{
						BaseScore: 6.4,
						Vector:    "AV:N/AC:M/Au:M/C:C/I:N/A:P",
					},
					Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
					Url:    "https://nvd.nist.gov/vuln/detail/CVE-1234-567",
				},
			},
			expected: &storage.EmbeddedVulnerability{
				Cvss:         8.2,
				ScoreVersion: storage.EmbeddedVulnerability_V3,
				Link:         "https://access.redhat.com/security/cve/CVE-1234-567",
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
						Url:    "https://access.redhat.com/security/cve/CVE-1234-567",
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
			},
		},
		{
			name: "CVSS 2 parse error",
			cvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
				{
					V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
						BaseScore: 8.2,
						Vector:    "CVSS:2.0/AV:A/AC:L/PR:N/UI:N/S:C/C:L/I:N/A:Q",
					},
					Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_RED_HAT,
					Url:    "https://access.redhat.com/security/cve/CVE-1234-567",
				},
			},
			wantErr: true,
		},
		{
			name: "CVSS 3.1 parse error",
			cvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
				{
					V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
						BaseScore: 8.2,
						Vector:    "CVSS:3.1/AV:A/AC:L/PR:N/UI:N/S:C/C:L/I:N/A:Q",
					},
					Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_RED_HAT,
					Url:    "https://access.redhat.com/security/cve/CVE-1234-567",
				},
			},
			wantErr: true,
		},
		{
			name: "CVSS 2 only",
			cvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
				{
					V2: &v4.VulnerabilityReport_Vulnerability_CVSS_V2{
						BaseScore: 6.4,
						Vector:    "AV:N/AC:M/Au:M/C:C/I:N/A:P",
					},
					Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_OSV,
					Url:    "https://osv.dev/vulnerability/CVE-1234-567",
				},
			},
			expected: &storage.EmbeddedVulnerability{
				Cvss:         6.4,
				ScoreVersion: storage.EmbeddedVulnerability_V2,
				Link:         "https://osv.dev/vulnerability/CVE-1234-567",
				CvssV2: &storage.CVSSV2{
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
				CvssMetrics: []*storage.CVSSScore{
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
						Source: storage.Source_SOURCE_OSV,
						Url:    "https://osv.dev/vulnerability/CVE-1234-567",
					},
				},
			},
		},
		{
			name: "CVSS 3.0 only",
			cvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
				{
					V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
						BaseScore: 8.2,
						Vector:    "CVSS:3.0/AV:A/AC:L/PR:N/UI:N/S:C/C:L/I:N/A:H",
					},
					Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_RED_HAT,
					Url:    "https://access.redhat.com/security/cve/CVE-1234-567",
				},
				{
					V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
						BaseScore: 5.0,
						Vector:    "CVSS:3.0/AV:N/AC:L/PR:L/UI:N/S:U/C:N/I:L/A:N",
					},
					Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
					Url:    "https://nvd.nist.gov/vuln/detail/CVE-1234-567",
				},
			},
			expected: &storage.EmbeddedVulnerability{
				Cvss:         8.2,
				ScoreVersion: storage.EmbeddedVulnerability_V3,
				Link:         "https://access.redhat.com/security/cve/CVE-1234-567",
				CvssV3: &storage.CVSSV3{
					Vector:              "CVSS:3.0/AV:A/AC:L/PR:N/UI:N/S:C/C:L/I:N/A:H",
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
								Vector:              "CVSS:3.0/AV:A/AC:L/PR:N/UI:N/S:C/C:L/I:N/A:H",
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
						Url:    "https://access.redhat.com/security/cve/CVE-1234-567",
					},
					{
						CvssScore: &storage.CVSSScore_Cvssv3{
							Cvssv3: &storage.CVSSV3{
								Vector:              "CVSS:3.0/AV:N/AC:L/PR:L/UI:N/S:U/C:N/I:L/A:N",
								ExploitabilityScore: 2.8,
								ImpactScore:         1.4,
								AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
								AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
								PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_LOW,
								UserInteraction:     storage.CVSSV3_UI_NONE,
								Scope:               storage.CVSSV3_UNCHANGED,
								Confidentiality:     storage.CVSSV3_IMPACT_NONE,
								Integrity:           storage.CVSSV3_IMPACT_LOW,
								Availability:        storage.CVSSV3_IMPACT_NONE,
								Score:               5.0,
								Severity:            storage.CVSSV3_MEDIUM,
							},
						},
						Source: storage.Source_SOURCE_NVD,
						Url:    "https://nvd.nist.gov/vuln/detail/CVE-1234-567",
					},
				},
			},
		},
		{
			name: "CVSS 3.1 only",
			cvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
				{
					V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
						BaseScore: 8.2,
						Vector:    "CVSS:3.1/AV:A/AC:L/PR:N/UI:N/S:C/C:L/I:N/A:H",
					},
					Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_OSV,
					Url:    "https://osv.dev/vulnerability/CVE-1234-567",
				},
				{
					V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
						BaseScore: 5.0,
						Vector:    "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:N/I:L/A:N",
					},
					Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
					Url:    "https://nvd.nist.gov/vuln/detail/CVE-1234-567",
				},
			},
			expected: &storage.EmbeddedVulnerability{
				Cvss:         8.2,
				ScoreVersion: storage.EmbeddedVulnerability_V3,
				Link:         "https://osv.dev/vulnerability/CVE-1234-567",
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
						Source: storage.Source_SOURCE_OSV, // Updated to match the correct source
						Url:    "https://osv.dev/vulnerability/CVE-1234-567",
					},
					{
						CvssScore: &storage.CVSSScore_Cvssv3{
							Cvssv3: &storage.CVSSV3{
								Vector:              "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:N/I:L/A:N",
								ExploitabilityScore: 2.8,
								ImpactScore:         1.4,
								AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
								AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
								PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_LOW,
								UserInteraction:     storage.CVSSV3_UI_NONE,
								Scope:               storage.CVSSV3_UNCHANGED,
								Confidentiality:     storage.CVSSV3_IMPACT_NONE,
								Integrity:           storage.CVSSV3_IMPACT_LOW,
								Availability:        storage.CVSSV3_IMPACT_NONE,
								Score:               5.0,
								Severity:            storage.CVSSV3_MEDIUM,
							},
						},
						Source: storage.Source_SOURCE_NVD,
						Url:    "https://nvd.nist.gov/vuln/detail/CVE-1234-567",
					},
				},
			},
		}}
	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			vuln := &storage.EmbeddedVulnerability{}
			err := setScoresAndScoreVersions(vuln, testcase.cvssMetrics)
			if testcase.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			protoassert.Equal(t, testcase.expected, vuln)
		})
	}
}

func TestMaybeOverwriteSeverity(t *testing.T) {
	testcases := []struct {
		name     string
		vuln     *storage.EmbeddedVulnerability
		expected storage.VulnerabilitySeverity
	}{
		{
			name:     "low no overwrite",
			expected: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
			vuln: &storage.EmbeddedVulnerability{
				Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
				CvssV3: &storage.CVSSV3{
					Severity: storage.CVSSV3_MEDIUM,
				},
				CvssV2: &storage.CVSSV2{
					Severity: storage.CVSSV2_HIGH,
				},
			},
		},
		{
			name:     "moderate no overwrite",
			expected: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
			vuln: &storage.EmbeddedVulnerability{
				Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
				CvssV3: &storage.CVSSV3{
					Severity: storage.CVSSV3_MEDIUM,
				},
				CvssV2: &storage.CVSSV2{
					Severity: storage.CVSSV2_HIGH,
				},
			},
		},
		{
			name:     "important no overwrite",
			expected: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
			vuln: &storage.EmbeddedVulnerability{
				Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
				CvssV3: &storage.CVSSV3{
					Severity: storage.CVSSV3_MEDIUM,
				},
				CvssV2: &storage.CVSSV2{
					Severity: storage.CVSSV2_HIGH,
				},
			},
		},
		{
			name:     "critical no overwrite",
			expected: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
			vuln: &storage.EmbeddedVulnerability{
				Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
				CvssV3: &storage.CVSSV3{
					Severity: storage.CVSSV3_MEDIUM,
				},
				CvssV2: &storage.CVSSV2{
					Severity: storage.CVSSV2_HIGH,
				},
			},
		},
		{
			name:     "CVSSv3 overwrite",
			expected: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
			vuln: &storage.EmbeddedVulnerability{
				Severity: storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY,
				CvssV3: &storage.CVSSV3{
					Severity: storage.CVSSV3_MEDIUM,
				},
				CvssV2: &storage.CVSSV2{
					Severity: storage.CVSSV2_HIGH,
				},
			},
		},
		{
			name:     "CVSSv2 overwrite",
			expected: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
			vuln: &storage.EmbeddedVulnerability{
				Severity: storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY,
				CvssV2: &storage.CVSSV2{
					Severity: storage.CVSSV2_HIGH,
				},
			},
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			maybeOverwriteSeverity(testcase.vuln)
			assert.Equal(t, testcase.expected, testcase.vuln.Severity)
		})
	}
}

func TestParsePackageDB(t *testing.T) {
	testcases := []struct {
		packageDB          string
		expectedSourceType storage.SourceType
		expectedLocation   string
	}{
		{
			packageDB:          "var/lib/dpkg/status",
			expectedSourceType: storage.SourceType_OS,
			expectedLocation:   "var/lib/dpkg/status",
		},
		{
			packageDB:          "sqlite:var/lib/rpm/rpmdb.sqlite",
			expectedSourceType: storage.SourceType_OS,
			expectedLocation:   "var/lib/rpm/rpmdb.sqlite",
		},
		{
			packageDB:          "go:usr/local/bin/scanner",
			expectedSourceType: storage.SourceType_GO,
			expectedLocation:   "usr/local/bin/scanner",
		},
		{
			packageDB:          "file:pkg.jar",
			expectedSourceType: storage.SourceType_JAVA,
			expectedLocation:   "pkg.jar",
		},
		{
			packageDB:          "jar:pkg.jar",
			expectedSourceType: storage.SourceType_JAVA,
			expectedLocation:   "pkg.jar",
		},
		{
			packageDB:          "maven:pkg.jar",
			expectedSourceType: storage.SourceType_JAVA,
			expectedLocation:   "pkg.jar",
		},
		{
			packageDB:          "nodejs:package.json",
			expectedSourceType: storage.SourceType_NODEJS,
			expectedLocation:   "package.json",
		},
		{
			packageDB:          "python:hello/.egg-info",
			expectedSourceType: storage.SourceType_PYTHON,
			expectedLocation:   "hello/.egg-info",
		},
		{
			packageDB:          "ruby:opt/specifications/howdy.gemspec",
			expectedSourceType: storage.SourceType_RUBY,
			expectedLocation:   "opt/specifications/howdy.gemspec",
		},
		{
			packageDB:          "h:e:llo",
			expectedSourceType: storage.SourceType_OS,
			expectedLocation:   "h:e:llo",
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.packageDB, func(t *testing.T) {
			source, location := ParsePackageDB(testcase.packageDB)
			assert.Equal(t, testcase.expectedSourceType, source)
			assert.Equal(t, testcase.expectedLocation, location)
		})
	}
}

func TestOS(t *testing.T) {
	testcases := []struct {
		expected string
		report   *v4.VulnerabilityReport
	}{
		{
			expected: "rhel:9",
			report: &v4.VulnerabilityReport{
				Contents: &v4.Contents{
					Distributions: []*v4.Distribution{
						{
							Did:       "rhel",
							VersionId: "9",
							Version:   "9",
						},
					},
				},
			},
		},
		{
			expected: "ubuntu:22.04",
			report: &v4.VulnerabilityReport{
				Contents: &v4.Contents{
					Distributions: []*v4.Distribution{
						{
							Did:       "ubuntu",
							VersionId: "22.04",
							Version:   "22.04 (Jammy)",
						},
					},
				},
			},
		},
		{
			expected: "debian:12",
			report: &v4.VulnerabilityReport{
				Contents: &v4.Contents{
					Distributions: []*v4.Distribution{
						{
							Did:       "debian",
							VersionId: "12",
							Version:   "12 (bookworm)",
						},
					},
				},
			},
		},
		{
			expected: "alpine:3.18",
			report: &v4.VulnerabilityReport{
				Contents: &v4.Contents{
					Distributions: []*v4.Distribution{
						{
							Did:       "alpine",
							VersionId: "3.18",
							Version:   "3.18",
						},
					},
				},
			},
		},
		{
			expected: "unknown",
			report: &v4.VulnerabilityReport{
				Contents: &v4.Contents{
					Distributions: []*v4.Distribution{
						{
							Did:       "alpine",
							VersionId: "3.18",
							Version:   "3.18",
						},
						{
							Did: "idk",
						},
					},
				},
			},
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.expected, func(t *testing.T) {
			name := os(testcase.report)
			assert.Equal(t, testcase.expected, name)
		})
	}
}

func TestNotes(t *testing.T) {
	testcases := []struct {
		os       string
		report   *v4.VulnerabilityReport
		expected []storage.ImageScan_Note
	}{
		{
			os: "unknown",
			report: &v4.VulnerabilityReport{
				Notes: []v4.VulnerabilityReport_Note{v4.VulnerabilityReport_NOTE_OS_UNKNOWN},
			},
			expected: []storage.ImageScan_Note{storage.ImageScan_OS_UNAVAILABLE, storage.ImageScan_PARTIAL_SCAN_DATA},
		},
		{
			os: "debian:8",
			report: &v4.VulnerabilityReport{
				Notes: []v4.VulnerabilityReport_Note{v4.VulnerabilityReport_NOTE_OS_UNSUPPORTED},
			},
			expected: []storage.ImageScan_Note{storage.ImageScan_OS_CVES_UNAVAILABLE, storage.ImageScan_PARTIAL_SCAN_DATA},
		},
		{
			os:       "rhel:9",
			report:   &v4.VulnerabilityReport{},
			expected: []storage.ImageScan_Note{},
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.os, func(t *testing.T) {
			notes := notes(testcase.report)
			assert.ElementsMatch(t, testcase.expected, notes)
		})
	}
}
