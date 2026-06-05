package scannerv4

import (
	"fmt"
	"testing"
	"time"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const scannerVersion = "indexer=4.8.3"

func TestNoPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		imageScan(nil, nil, "")

		report := &v4.VulnerabilityReport{}
		imageScan(nil, report, scannerVersion)

		report.Contents = &v4.Contents{}
		imageScan(nil, report, scannerVersion)

		report.Contents.Packages = map[string]*v4.Package{}
		imageScan(nil, report, scannerVersion)

		report.Contents.Packages = map[string]*v4.Package{"1": {Id: "1"}}
		imageScan(nil, report, scannerVersion)

		report.PackageVulnerabilities = map[string]*v4.StringList{}
		imageScan(nil, report, scannerVersion)

		report.PackageVulnerabilities["1"] = &v4.StringList{}
		imageScan(nil, report, scannerVersion)

		report.PackageVulnerabilities["1"].Values = []string{}
		imageScan(nil, report, scannerVersion)

		report.PackageVulnerabilities["1"].Values = []string{"CVE1"}
		imageScan(nil, report, scannerVersion)
	})
}

func TestConvert(t *testing.T) {
	protoNow, err := protocompat.ConvertTimeToTimestampOrError(time.Now())
	require.NoError(t, err)

	testcases := []struct {
		name     string
		metadata *storage.ImageMetadata
		report   *v4.VulnerabilityReport
		expected *storage.ImageScan
	}{
		{
			name: "basic",
			metadata: &storage.ImageMetadata{
				V2: &storage.V2Metadata{},
				V1: &storage.V1Metadata{
					Layers: []*storage.ImageLayer{
						{Empty: false},
					},
				},
				LayerShas: []string{"hash"},
			},
			report: &v4.VulnerabilityReport{
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
					Distributions: map[string]*v4.Distribution{
						"0": {
							Did:       "rhel",
							VersionId: "9",
						},
					},
					Packages: map[string]*v4.Package{
						"1": {
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
						FixedDate:          protoNow,
						NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_IMPORTANT,
					},
				},
				PackageVulnerabilities: map[string]*v4.StringList{
					"1": {
						Values: []string{"CVE1-ID"},
					},
				},
			},
			expected: &storage.ImageScan{
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Name:          "my-java-pkg",
						Version:       "1.2.3",
						Source:        storage.SourceType_JAVA,
						Location:      "opt/java/pkg.jar",
						HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve:                   "CVE1-Name",
								VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
								Severity:              storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
								SetFixedBy:            &storage.EmbeddedVulnerability_FixedBy{FixedBy: "v99"},
								FixAvailableTimestamp: protoNow,
							},
						},
					},
				},
				OperatingSystem: "rhel:9",
			},
		},
		{
			name: "deprecated",
			metadata: &storage.ImageMetadata{
				V2: &storage.V2Metadata{},
				V1: &storage.V1Metadata{
					Layers: []*storage.ImageLayer{
						{Empty: false},
					},
				},
				LayerShas: []string{"hash"},
			},
			report: &v4.VulnerabilityReport{
				Contents: &v4.Contents{
					EnvironmentsDEPRECATED: map[string]*v4.Environment_List{
						"1": {
							Environments: []*v4.Environment{
								{
									PackageDb:    "maven:opt/java/pkg.jar",
									IntroducedIn: "hash",
								},
							},
						},
					},
					DistributionsDEPRECATED: []*v4.Distribution{
						{
							Did:       "rhel",
							VersionId: "9",
						},
					},
					PackagesDEPRECATED: []*v4.Package{
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
			},
			expected: &storage.ImageScan{
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Name:          "my-java-pkg",
						Version:       "1.2.3",
						Source:        storage.SourceType_JAVA,
						Location:      "opt/java/pkg.jar",
						HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve:                   "CVE1-Name",
								VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
								Severity:              storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
								SetFixedBy:            &storage.EmbeddedVulnerability_FixedBy{FixedBy: "v99"},
								FixAvailableTimestamp: nil,
							},
						},
					},
				},
				OperatingSystem: "rhel:9",
			},
		},
		{
			name: "prefer non-deprecated",
			metadata: &storage.ImageMetadata{
				V2: &storage.V2Metadata{},
				V1: &storage.V1Metadata{
					Layers: []*storage.ImageLayer{
						{Empty: false},
					},
				},
				LayerShas: []string{"hash"},
			},
			report: &v4.VulnerabilityReport{
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
					EnvironmentsDEPRECATED: map[string]*v4.Environment_List{
						"1": {
							Environments: []*v4.Environment{
								{
									PackageDb:    "maven:opt/java/pkg2.jar",
									IntroducedIn: "hash",
								},
							},
						},
					},
					Distributions: map[string]*v4.Distribution{
						"0": {
							Did:       "rhel",
							VersionId: "9",
						},
					},
					DistributionsDEPRECATED: []*v4.Distribution{
						{
							Did:       "rhel",
							VersionId: "10",
						},
					},
					Packages: map[string]*v4.Package{
						"1": {
							Id:      "1",
							Name:    "my-java-pkg",
							Version: "1.2.3",
						},
					},
					PackagesDEPRECATED: []*v4.Package{
						{
							Id:      "1",
							Name:    "my-java-pkg",
							Version: "1.2.4",
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
			},
			expected: &storage.ImageScan{
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
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			got := imageScan(tc.metadata, tc.report, scannerVersion)
			protoassert.SlicesEqual(t, tc.expected.GetComponents(), got.GetComponents())
			assert.Equal(t, tc.expected.GetOperatingSystem(), got.GetOperatingSystem())
			assert.Equal(t, scannerVersion, got.GetScannerVersion())
		})
	}
}

func TestComponents(t *testing.T) {
	testcases := []struct {
		name     string
		dedupe   bool
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
					Packages: map[string]*v4.Package{
						"1": {
							Id:      "1",
							Name:    "glib2",
							Version: "2.68.4-14.el9",
						},
					},
					Distributions: map[string]*v4.Distribution{
						"1": {
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
					Packages: map[string]*v4.Package{
						"1": {
							Id:      "1",
							Name:    "glib2",
							Version: "2.68.4-14.el9",
						},
					},
					Distributions: map[string]*v4.Distribution{
						"1": {
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
		{
			name: "basic no vulns deprecated",
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
					PackagesDEPRECATED: []*v4.Package{
						{
							Id:      "1",
							Name:    "glib2",
							Version: "2.68.4-14.el9",
						},
					},
					DistributionsDEPRECATED: []*v4.Distribution{
						{
							Id:        "1",
							Did:       "rhel",
							VersionId: "9",
						},
					},
					EnvironmentsDEPRECATED: map[string]*v4.Environment_List{
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
			name:   "RHCC source+binary+ancestry kept without dedupe feature flag",
			dedupe: false,
			metadata: &storage.ImageMetadata{
				V1: &storage.V1Metadata{Digest: "d"},
			},
			report: &v4.VulnerabilityReport{
				Contents: &v4.Contents{
					Packages: map[string]*v4.Package{
						"src-1": {
							Id:      "src-1",
							Name:    "ubi9/ubi-micro",
							Version: "1779858857",
							Kind:    "source",
						},
						"bin-1": {
							Id:      "bin-1",
							Name:    "ubi9/ubi-micro",
							Version: "1779858857",
							Kind:    "binary",
							Source:  &v4.Package{Id: "src-1", Name: "ubi9/ubi-micro", Kind: "source"},
						},
						"anc-1": {
							Id:      "anc-1",
							Name:    "ubi9/ubi-micro",
							Version: "1779858857",
							Kind:    "ancestry",
							Source:  &v4.Package{Id: "src-1", Name: "ubi9/ubi-micro", Kind: "source"},
						},
					},
					Environments: map[string]*v4.Environment_List{
						"bin-1": {Environments: []*v4.Environment{{PackageDb: "root/buildinfo/labels.json"}}},
						"src-1": {Environments: []*v4.Environment{{PackageDb: "root/buildinfo/labels.json"}}},
						"anc-1": {Environments: []*v4.Environment{{PackageDb: "root/buildinfo/labels.json"}}},
					},
				},
			},
			expected: []*storage.EmbeddedImageScanComponent{
				{
					Name:     "ubi9/ubi-micro",
					Version:  "1779858857",
					Source:   storage.SourceType_OS,
					Location: "root/buildinfo/labels.json",
				},
				{
					Name:     "ubi9/ubi-micro",
					Version:  "1779858857",
					Source:   storage.SourceType_OS,
					Location: "root/buildinfo/labels.json",
				},
				{
					Name:     "ubi9/ubi-micro",
					Version:  "1779858857",
					Source:   storage.SourceType_OS,
					Location: "root/buildinfo/labels.json",
				},
			},
		},
		{
			name:   "RHCC source+binary+ancestry only keeps binary",
			dedupe: true,
			metadata: &storage.ImageMetadata{
				V1: &storage.V1Metadata{Digest: "d"},
			},
			report: &v4.VulnerabilityReport{
				Contents: &v4.Contents{
					Packages: map[string]*v4.Package{
						"src-1": {
							Id:      "src-1",
							Name:    "ubi9/ubi-micro",
							Version: "1779858857",
							Kind:    "source",
						},
						"bin-1": {
							Id:      "bin-1",
							Name:    "ubi9/ubi-micro",
							Version: "1779858857",
							Kind:    "binary",
							Source:  &v4.Package{Id: "src-1", Name: "ubi9/ubi-micro", Kind: "source"},
						},
						"anc-1": {
							Id:      "anc-1",
							Name:    "ubi9/ubi-micro",
							Version: "1779858857",
							Kind:    "ancestry",
							Source:  &v4.Package{Id: "src-1", Name: "ubi9/ubi-micro", Kind: "source"},
						},
					},
					Environments: map[string]*v4.Environment_List{
						"bin-1": {Environments: []*v4.Environment{{PackageDb: "root/buildinfo/labels.json"}}},
						"src-1": {Environments: []*v4.Environment{{PackageDb: "root/buildinfo/labels.json"}}},
						"anc-1": {Environments: []*v4.Environment{{PackageDb: "root/buildinfo/labels.json"}}},
					},
				},
			},
			expected: []*storage.EmbeddedImageScanComponent{
				{
					Name:     "ubi9/ubi-micro",
					Version:  "1779858857",
					Source:   storage.SourceType_OS,
					Location: "root/buildinfo/labels.json",
				},
			},
		},
		{
			name:   "source with different name from binary is still filtered",
			dedupe: true,
			metadata: &storage.ImageMetadata{
				V1: &storage.V1Metadata{Digest: "d"},
			},
			report: &v4.VulnerabilityReport{
				Contents: &v4.Contents{
					Packages: map[string]*v4.Package{
						"src-1": {
							Id:      "src-1",
							Name:    "rhel-els-container",
							Version: "9.4-847",
							Kind:    "source",
						},
						"bin-1": {
							Id:      "bin-1",
							Name:    "rhel-els",
							Version: "9.4-847",
							Kind:    "binary",
							Source:  &v4.Package{Id: "src-1", Name: "rhel-els-container", Kind: "source"},
						},
					},
					Environments: map[string]*v4.Environment_List{
						"bin-1": {Environments: []*v4.Environment{{PackageDb: "root/buildinfo/Dockerfile-rhel-els"}}},
						"src-1": {Environments: []*v4.Environment{{PackageDb: "root/buildinfo/Dockerfile-rhel-els"}}},
					},
				},
			},
			expected: []*storage.EmbeddedImageScanComponent{
				{
					Name:     "rhel-els",
					Version:  "9.4-847",
					Source:   storage.SourceType_OS,
					Location: "root/buildinfo/Dockerfile-rhel-els",
				},
			},
		},
		{
			name:   "source+ancestry without binary retains source",
			dedupe: true,
			metadata: &storage.ImageMetadata{
				V1: &storage.V1Metadata{Digest: "d"},
			},
			report: &v4.VulnerabilityReport{
				Contents: &v4.Contents{
					Packages: map[string]*v4.Package{
						"src-1": {
							Id:      "src-1",
							Name:    "ubi9/ubi-micro",
							Version: "1779858857",
							Kind:    "source",
						},
						"anc-1": {
							Id:      "anc-1",
							Name:    "ubi9/ubi-micro",
							Version: "1779858857",
							Kind:    "ancestry",
							Source:  &v4.Package{Id: "src-1", Name: "ubi9/ubi-micro", Kind: "source"},
						},
					},
					Environments: map[string]*v4.Environment_List{
						"src-1": {Environments: []*v4.Environment{{PackageDb: "root/buildinfo/labels.json"}}},
						"anc-1": {Environments: []*v4.Environment{{PackageDb: "root/buildinfo/labels.json"}}},
					},
				},
			},
			expected: []*storage.EmbeddedImageScanComponent{
				{
					Name:     "ubi9/ubi-micro",
					Version:  "1779858857",
					Source:   storage.SourceType_OS,
					Location: "root/buildinfo/labels.json",
				},
			},
		},
		{
			name:   "standalone source package not referenced by binary is kept",
			dedupe: true,
			metadata: &storage.ImageMetadata{
				V1: &storage.V1Metadata{Digest: "d"},
			},
			report: &v4.VulnerabilityReport{
				Contents: &v4.Contents{
					Packages: map[string]*v4.Package{
						"src-1": {
							Id:      "src-1",
							Name:    "standalone-src",
							Version: "1.0",
							Kind:    "source",
						},
					},
					Environments: map[string]*v4.Environment_List{
						"src-1": {Environments: []*v4.Environment{{PackageDb: "sqlite:var/lib/rpm"}}},
					},
				},
			},
			expected: []*storage.EmbeddedImageScanComponent{
				{
					Name:     "standalone-src",
					Version:  "1.0",
					Source:   storage.SourceType_OS,
					Location: "var/lib/rpm",
				},
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			testutils.MustUpdateFeature(t, features.ScannerV4Dedupe, tc.dedupe)
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
			if result.GetEpssProbability() != testcase.expected.GetEpssProbability() || result.GetEpssPercentile() != testcase.expected.GetEpssPercentile() {
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
			assert.Equal(t, testcase.expected, testcase.vuln.GetSeverity())
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
					Distributions: map[string]*v4.Distribution{
						"-1": {
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
					Distributions: map[string]*v4.Distribution{
						"0": {
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
					Distributions: map[string]*v4.Distribution{
						"1": {
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
					Distributions: map[string]*v4.Distribution{
						"3": {
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
					Distributions: map[string]*v4.Distribution{
						"4": {
							Did:       "alpine",
							VersionId: "3.18",
							Version:   "3.18",
						},
						"idk": {
							Did: "idk",
						},
					},
				},
			},
		},
		{
			expected: "rhel:10",
			report: &v4.VulnerabilityReport{
				Contents: &v4.Contents{
					DistributionsDEPRECATED: []*v4.Distribution{
						{
							Did:       "rhel",
							VersionId: "10",
							Version:   "10",
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

func TestEnvOS(t *testing.T) {
	testcases := []struct {
		expected string
		env      *v4.Environment
		report   *v4.VulnerabilityReport
	}{
		{
			expected: "",
			env:      nil,
			report:   nil,
		},
		{
			expected: "",
			env:      &v4.Environment{DistributionId: "-1"},
			report:   nil,
		},
		{
			expected: "",
			env:      nil,
			report: &v4.VulnerabilityReport{
				Contents: &v4.Contents{
					Distributions: map[string]*v4.Distribution{
						"-1": {
							Did:       "rhel",
							VersionId: "9",
						},
					},
				},
			},
		},
		{
			expected: "",
			env:      &v4.Environment{DistributionId: "noexist"},
			report: &v4.VulnerabilityReport{
				Contents: &v4.Contents{
					Distributions: map[string]*v4.Distribution{
						"-1": {
							Did:       "rhel",
							VersionId: "9",
						},
					},
				},
			},
		},
		{
			expected: "",
			env:      &v4.Environment{DistributionId: "-1"},
			report: &v4.VulnerabilityReport{
				Contents: &v4.Contents{
					Distributions: map[string]*v4.Distribution{
						"-1": {
							Did: "rhel",
							// VersionId missing
						},
					},
				},
			},
		},
		{
			expected: "",
			env:      &v4.Environment{DistributionId: "-1"},
			report: &v4.VulnerabilityReport{
				Contents: &v4.Contents{
					Distributions: map[string]*v4.Distribution{
						"-1": {
							// Did missing
							VersionId: "9",
						},
					},
				},
			},
		},
		{
			expected: "ubuntu:22.04",
			env:      &v4.Environment{DistributionId: "0"},
			report: &v4.VulnerabilityReport{
				Contents: &v4.Contents{
					Distributions: map[string]*v4.Distribution{
						"0": {
							Did:       "ubuntu",
							VersionId: "22.04",
						},
					},
				},
			},
		},
		{
			expected: "alpine:3.18",
			env:      &v4.Environment{DistributionId: "4"},
			report: &v4.VulnerabilityReport{
				Contents: &v4.Contents{
					Distributions: map[string]*v4.Distribution{
						"4": {
							Did:       "alpine",
							VersionId: "3.18",
						},
						"idk": {
							Did: "idk",
						},
					},
				},
			},
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.expected, func(t *testing.T) {
			name := envOS(testcase.env, testcase.report)
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

// TestVulnDataSource
//
// If this test fails due to a datasource format change
// a Central DB migration may be needed convert stored values to the
// new format.
func TestVulnDataSource(t *testing.T) {
	testcases := []struct {
		expected string
		os       string
		ccVuln   *v4.VulnerabilityReport_Vulnerability
	}{
		{
			expected: "",
			os:       "",
			ccVuln:   nil,
		},
		{
			expected: "",
			os:       "os",
			ccVuln:   nil,
		},
		{
			expected: "",
			os:       "os",
			ccVuln:   &v4.VulnerabilityReport_Vulnerability{},
		},
		{
			expected: "updater",
			os:       "",
			ccVuln: &v4.VulnerabilityReport_Vulnerability{
				Updater: "updater",
			},
		},
		{
			expected: "updater::os",
			os:       "os",
			ccVuln: &v4.VulnerabilityReport_Vulnerability{
				Updater: "updater",
			},
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.expected, func(t *testing.T) {
			name := vulnDataSource(testcase.ccVuln, testcase.os)
			assert.Equal(t, testcase.expected, name)
		})
	}
}

func TestVulnerabilities_DedupByCVEName(t *testing.T) {
	testutils.MustUpdateFeature(t, features.ScannerV4Dedupe, true)

	t.Run("duplicate CVEs are merged with highest severity", func(t *testing.T) {
		vulnMap := map[string]*v4.VulnerabilityReport_Vulnerability{
			"a": {
				Id: "a", Name: "CVE-2024-1", Description: "short",
				NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_LOW,
				FixedInVersion:     "1.0.0",
			},
			"b": {
				Id: "b", Name: "CVE-2024-1", Description: "a much longer description",
				NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_CRITICAL,
				FixedInVersion:     "2.0.0",
			},
		}
		got := vulnerabilities(vulnMap, []string{"a", "b"}, "", "")
		require.Len(t, got, 1)
		assert.Equal(t, "CVE-2024-1", got[0].GetCve())
		// Fix primary is "b" (higher fix version by best-effort comparison): fix fields from "b".
		assert.Equal(t, "2.0.0", got[0].GetFixedBy())
		// Advisories are the same (both nil), so summary comes from scoring
		// primary ("b" — higher severity).
		assert.Equal(t, "a much longer description", got[0].GetSummary())
		assert.Equal(t, storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, got[0].GetSeverity())
	})

	t.Run("RHSA merge: newer advisory wins fix fields, same scoring", func(t *testing.T) {
		vulnMap := map[string]*v4.VulnerabilityReport_Vulnerability{
			"a": {
				Id: "a", Name: "CVE-2024-8176", Description: "stack overflow in libexpat",
				Advisory:           &v4.VulnerabilityReport_Advisory{Name: "RHSA-2025:3531", Link: "https://access.redhat.com/errata/RHSA-2025:3531"},
				NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_MODERATE,
				FixedInVersion:     "0:2.5.0-3.el9_5.3",
				Link:               "https://access.redhat.com/security/cve/CVE-2024-8176",
				CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
					{
						Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_RED_HAT,
						Url:    "https://access.redhat.com/security/cve/CVE-2024-8176",
						V3:     &v4.VulnerabilityReport_Vulnerability_CVSS_V3{BaseScore: 7.5, Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H"},
					},
				},
			},
			"b": {
				Id: "b", Name: "CVE-2024-8176", Description: "stack overflow in libexpat",
				Advisory:           &v4.VulnerabilityReport_Advisory{Name: "RHSA-2025:7444", Link: "https://access.redhat.com/errata/RHSA-2025:7444"},
				NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_MODERATE,
				FixedInVersion:     "0:2.5.0-5.el9_6",
				Link:               "https://access.redhat.com/security/cve/CVE-2024-8176",
				CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
					{
						Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_RED_HAT,
						Url:    "https://access.redhat.com/security/cve/CVE-2024-8176",
						V3:     &v4.VulnerabilityReport_Vulnerability_CVSS_V3{BaseScore: 7.5, Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H"},
					},
				},
			},
		}
		got := vulnerabilities(vulnMap, []string{"a", "b"}, "", "")
		require.Len(t, got, 1)
		assert.Equal(t, "CVE-2024-8176", got[0].GetCve())
		assert.Equal(t, "RHSA-2025:7444", got[0].GetAdvisory().GetName())
		assert.Equal(t, "0:2.5.0-5.el9_6", got[0].GetFixedBy())
	})

	t.Run("no advisory: more CVSS metrics wins scoring", func(t *testing.T) {
		vulnMap := map[string]*v4.VulnerabilityReport_Vulnerability{
			"a": {
				Id: "a", Name: "CVE-2026-33997", Description: "Off-by-one error",
				NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_MODERATE,
				Link:               "https://osv.dev/vulnerability/GHSA-pxq6-2prw-chj9",
				CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
					{
						Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_OSV,
						Url:    "https://osv.dev/vulnerability/GHSA-pxq6-2prw-chj9",
						V3:     &v4.VulnerabilityReport_Vulnerability_CVSS_V3{BaseScore: 6.8, Vector: "CVSS:3.1/AV:N/AC:H/PR:N/UI:R/S:U/C:H/I:H/A:N"},
					},
					{
						Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
						Url:    "https://nvd.nist.gov/vuln/detail/CVE-2026-33997",
						V3:     &v4.VulnerabilityReport_Vulnerability_CVSS_V3{BaseScore: 8.1, Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:N"},
					},
				},
			},
			"b": {
				Id: "b", Name: "CVE-2026-33997", Description: "Off-by-one error in github.com/docker/docker",
				NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_IMPORTANT,
				Link:               "https://nvd.nist.gov/vuln/detail/CVE-2026-33997",
				CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
					{
						Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
						Url:    "https://nvd.nist.gov/vuln/detail/CVE-2026-33997",
						V3:     &v4.VulnerabilityReport_Vulnerability_CVSS_V3{BaseScore: 8.1, Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:N"},
					},
				},
			},
		}
		got := vulnerabilities(vulnMap, []string{"a", "b"}, "", "")
		require.Len(t, got, 1)
		assert.Equal(t, "CVE-2026-33997", got[0].GetCve())
		// A wins scoring (2 metrics > 1): summary, severity from A.
		assert.Equal(t, "Off-by-one error", got[0].GetSummary())
		assert.Len(t, got[0].GetCvssMetrics(), 2)
	})

	t.Run("no advisory: entry with fix wins over entry without", func(t *testing.T) {
		vulnMap := map[string]*v4.VulnerabilityReport_Vulnerability{
			"a": {
				Id: "a", Name: "CVE-2026-34040", Description: "AuthZ plugin bypass",
				NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_IMPORTANT,
				FixedInVersion:     "29.3.1",
				Link:               "https://osv.dev/vulnerability/GHSA-x744-4wpc-v9h2",
				CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
					{
						Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_OSV,
						Url:    "https://osv.dev/vulnerability/GHSA-x744-4wpc-v9h2",
						V3:     &v4.VulnerabilityReport_Vulnerability_CVSS_V3{BaseScore: 8.8, Vector: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:C/C:H/I:H/A:H"},
					},
					{
						Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
						Url:    "https://nvd.nist.gov/vuln/detail/CVE-2026-34040",
						V3:     &v4.VulnerabilityReport_Vulnerability_CVSS_V3{BaseScore: 7.8, Vector: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H"},
					},
				},
			},
			"b": {
				Id: "b", Name: "CVE-2026-34040", Description: "AuthZ plugin bypass in github.com/docker/docker",
				NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_IMPORTANT,
				Link:               "https://nvd.nist.gov/vuln/detail/CVE-2026-34040",
				CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
					{
						Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
						Url:    "https://nvd.nist.gov/vuln/detail/CVE-2026-34040",
						V3:     &v4.VulnerabilityReport_Vulnerability_CVSS_V3{BaseScore: 7.8, Vector: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H"},
					},
				},
			},
		}
		got := vulnerabilities(vulnMap, []string{"a", "b"}, "", "")
		require.Len(t, got, 1)
		assert.Equal(t, "CVE-2026-34040", got[0].GetCve())
		assert.Equal(t, "29.3.1", got[0].GetFixedBy())
		// A wins scoring (2 metrics > 1).
		assert.Equal(t, "AuthZ plugin bypass", got[0].GetSummary())
		assert.Len(t, got[0].GetCvssMetrics(), 2)
	})

	t.Run("no advisory: same metric count and severity, higher CVSS base score wins", func(t *testing.T) {
		// Same number of CVSS metrics, same severity — the higher base
		// score breaks the tie and its scoring fields are selected.
		vulnMap := map[string]*v4.VulnerabilityReport_Vulnerability{
			"a": {
				Id: "a", Name: "CVE-2024-9999", Description: "lower score entry",
				NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_IMPORTANT,
				FixedInVersion:     "1.0.0",
				Link:               "https://example.com/low",
				CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
					{
						Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
						Url:    "https://nvd.nist.gov/vuln/detail/CVE-2024-9999",
						V3:     &v4.VulnerabilityReport_Vulnerability_CVSS_V3{BaseScore: 5.9, Vector: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:N/A:H"},
					},
				},
			},
			"b": {
				Id: "b", Name: "CVE-2024-9999", Description: "higher score entry",
				NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_IMPORTANT,
				FixedInVersion:     "1.0.0",
				Link:               "https://example.com/high",
				CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
					{
						Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
						Url:    "https://nvd.nist.gov/vuln/detail/CVE-2024-9999",
						V3:     &v4.VulnerabilityReport_Vulnerability_CVSS_V3{BaseScore: 7.5, Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H"},
					},
				},
			},
		}
		got := vulnerabilities(vulnMap, []string{"a", "b"}, "", "")
		require.Len(t, got, 1)
		assert.Equal(t, "CVE-2024-9999", got[0].GetCve())
		// B wins scoring (higher base score: 7.5 > 5.9).
		assert.Equal(t, "higher score entry", got[0].GetSummary())
		assert.InDelta(t, 7.5, got[0].GetCvss(), 0.01)
	})

	t.Run("duplicate RHSA names are deduped", func(t *testing.T) {
		vulnMap := map[string]*v4.VulnerabilityReport_Vulnerability{
			"a": {Id: "a", Name: "RHSA-2024:100"},
			"b": {Id: "b", Name: "RHSA-2024:100"},
		}
		got := vulnerabilities(vulnMap, []string{"a", "b"}, "", "")
		assert.Len(t, got, 1)
	})

	t.Run("no duplicates passes through", func(t *testing.T) {
		vulnMap := map[string]*v4.VulnerabilityReport_Vulnerability{
			"a": {Id: "a", Name: "CVE-2024-1"},
			"b": {Id: "b", Name: "CVE-2024-2"},
		}
		got := vulnerabilities(vulnMap, []string{"a", "b"}, "", "")
		assert.Len(t, got, 2)
	})

	t.Run("feature flag off disables dedupe", func(t *testing.T) {
		testutils.MustUpdateFeature(t, features.ScannerV4Dedupe, false)
		vulnMap := map[string]*v4.VulnerabilityReport_Vulnerability{
			"a": {Id: "a", Name: "CVE-2024-1"},
			"b": {Id: "b", Name: "CVE-2024-1"},
		}
		got := vulnerabilities(vulnMap, []string{"a", "b"}, "", "")
		assert.Len(t, got, 2)
	})

	t.Run("higher fix version wins when all else equal", func(t *testing.T) {
		vulnMap := map[string]*v4.VulnerabilityReport_Vulnerability{
			"a": {
				Id: "a", Name: "CVE-2024-1",
				FixedInVersion: "1.2.3",
			},
			"b": {
				Id: "b", Name: "CVE-2024-1",
				FixedInVersion: "1.2.5",
			},
		}
		got := vulnerabilities(vulnMap, []string{"a", "b"}, "", "")
		require.Len(t, got, 1)
		assert.Equal(t, "1.2.5", got[0].GetFixedBy())
	})

	t.Run("three duplicates merge rolling", func(t *testing.T) {
		vulnMap := map[string]*v4.VulnerabilityReport_Vulnerability{
			"a": {
				Id: "a", Name: "CVE-2024-1",
				NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_LOW,
				FixedInVersion:     "1.0.0",
			},
			"b": {
				Id: "b", Name: "CVE-2024-1",
				NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_CRITICAL,
				FixedInVersion:     "2.0.0",
			},
			"c": {
				Id: "c", Name: "CVE-2024-1",
				NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_MODERATE,
				FixedInVersion:     "3.0.0",
			},
		}
		got := vulnerabilities(vulnMap, []string{"a", "b", "c"}, "", "")
		require.Len(t, got, 1)
		// Scoring: "b" wins (CRITICAL > MODERATE > LOW).
		assert.Equal(t, storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, got[0].GetSeverity())
		// Fix: "c" wins (3.0.0 > 2.0.0 > 1.0.0 by best-effort compare).
		assert.Equal(t, "3.0.0", got[0].GetFixedBy())
	})

	t.Run("equal scoring keeps first-seen", func(t *testing.T) {
		vulnMap := map[string]*v4.VulnerabilityReport_Vulnerability{
			"a": {
				Id: "a", Name: "CVE-2024-1", Description: "first",
				NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_LOW,
			},
			"b": {
				Id: "b", Name: "CVE-2024-1", Description: "second",
				NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_LOW,
			},
		}
		got := vulnerabilities(vulnMap, []string{"a", "b"}, "", "")
		require.Len(t, got, 1)
		// Equal scoring — first-seen ("a") keeps its summary.
		assert.Equal(t, "first", got[0].GetSummary())
	})

	t.Run("dst fix matches pkgFixedBy, src fix does not", func(t *testing.T) {
		vulnMap := map[string]*v4.VulnerabilityReport_Vulnerability{
			"a": {
				Id: "a", Name: "CVE-2024-1",
				FixedInVersion: "2.0.0",
			},
			"b": {
				Id: "b", Name: "CVE-2024-1",
				FixedInVersion: "1.5.0",
			},
		}
		// "a" matches pkgFixedBy, "b" does not — "a" keeps its fix.
		got := vulnerabilities(vulnMap, []string{"a", "b"}, "", "2.0.0")
		require.Len(t, got, 1)
		assert.Equal(t, "2.0.0", got[0].GetFixedBy())
	})

	t.Run("advisory wins with no fix version", func(t *testing.T) {
		vulnMap := map[string]*v4.VulnerabilityReport_Vulnerability{
			"a": {
				Id: "a", Name: "CVE-2024-1",
				Advisory:       &v4.VulnerabilityReport_Advisory{Name: "RHSA-2024:100"},
				FixedInVersion: "1.0.0",
			},
			"b": {
				Id: "b", Name: "CVE-2024-1",
				Advisory: &v4.VulnerabilityReport_Advisory{Name: "RHSA-2024:200"},
			},
		}
		got := vulnerabilities(vulnMap, []string{"a", "b"}, "", "")
		require.Len(t, got, 1)
		// "b" wins on advisory — fix version is empty from "b".
		assert.Equal(t, "RHSA-2024:200", got[0].GetAdvisory().GetName())
		assert.Equal(t, "", got[0].GetFixedBy())
	})
}

func TestCompareAdvisories(t *testing.T) {
	testcases := map[string]struct {
		a, b     *storage.Advisory
		expected int
	}{
		"both nil":                    {nil, nil, 0},
		"a nil":                       {nil, &storage.Advisory{Name: "RHSA-2024:100"}, -1},
		"b nil":                       {&storage.Advisory{Name: "RHSA-2024:100"}, nil, 1},
		"same":                        {&storage.Advisory{Name: "RHSA-2024:100"}, &storage.Advisory{Name: "RHSA-2024:100"}, 0},
		"same year different number":  {&storage.Advisory{Name: "RHSA-2024:100"}, &storage.Advisory{Name: "RHSA-2024:200"}, -1},
		"same year numeric not lex":   {&storage.Advisory{Name: "RHSA-2024:100"}, &storage.Advisory{Name: "RHSA-2024:90"}, 1},
		"different years":             {&storage.Advisory{Name: "RHSA-2023:500"}, &storage.Advisory{Name: "RHSA-2024:100"}, -1},
		"parseable year over not":     {&storage.Advisory{Name: "RHSA-2024:100"}, &storage.Advisory{Name: "some-advisory"}, 1},
		"neither parseable lex order": {&storage.Advisory{Name: "AAA-advisory"}, &storage.Advisory{Name: "ZZZ-advisory"}, -1},
	}
	for name, tt := range testcases {
		t.Run(name, func(t *testing.T) {
			got := compareAdvisories(tt.a, tt.b)
			assert.Equal(t, tt.expected, got)
			reverse := compareAdvisories(tt.b, tt.a)
			assert.Equal(t, -tt.expected, reverse)
		})
	}
}

func TestCompareNumericSegments(t *testing.T) {
	testcases := map[string]struct {
		a, b     string
		expected int
	}{
		"equal":                            {"1.2.3", "1.2.3", 0},
		"semver less":                      {"1.2.3", "1.2.5", -1},
		"semver greater":                   {"2.0.0", "1.9.9", 1},
		"major differs":                    {"1.0.0", "2.0.0", -1},
		"rpm style":                        {"0:1.2.3-4.el8", "0:1.2.3-5.el8", -1},
		"rpm epoch differs":                {"0:1.2.3-4.el8", "1:1.2.3-4.el8", -1},
		"longer version wins":              {"1.2.3", "1.2.3.1", -1},
		"shorter version loses":            {"1.2.3.1", "1.2.3", 1},
		"debian style":                     {"1.2.3-4ubuntu5", "1.2.3-4ubuntu6", -1},
		"same numeric diff suffix":         {"1.2.3-alpha", "1.2.3-beta", -1},
		"more segments but smaller values": {"0.0.0.0.1", "1.0.0", -1},
		"fewer segments but larger values": {"2.0", "1.9.9.9.9", 1},
		"empty equal":                      {"", "", 0},
	}
	for name, tt := range testcases {
		t.Run(name, func(t *testing.T) {
			got := compareNumericSegments(tt.a, tt.b)
			assert.Equal(t, tt.expected, got)
			if tt.expected != 0 {
				reverse := compareNumericSegments(tt.b, tt.a)
				assert.Equal(t, -tt.expected, reverse)
			}
		})
	}
}
