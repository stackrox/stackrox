package scannerv4

import (
	"testing"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
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

	assert.Equal(t, expected.Components, actual.Components)
	assert.Equal(t, expected.OperatingSystem, actual.OperatingSystem)
}

func TestLocation(t *testing.T) {
	testcases := []struct {
		expected string
		env      *v4.Environment
	}{
		{
			expected: "var/lib/dpkg/status",
			env: &v4.Environment{
				PackageDb: "var/lib/dpkg/status",
			},
		},
		{
			expected: "var/lib/rpm/rpmdb.sqlite",
			env: &v4.Environment{
				PackageDb: "sqlite:var/lib/rpm/rpmdb.sqlite",
			},
		},
		{
			expected: "usr/local/bin/scanner",
			env: &v4.Environment{
				PackageDb: "go:usr/local/bin/scanner",
			},
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.expected, func(t *testing.T) {
			source := location(testcase.env)
			assert.Equal(t, testcase.expected, source)
		})
	}
}

func TestSourceType(t *testing.T) {
	testcases := []struct {
		expected storage.SourceType
		env      *v4.Environment
	}{
		{
			expected: storage.SourceType_OS,
			env: &v4.Environment{
				PackageDb: "sqlite:var/lib/rpm/rpmdb.sqlite",
			},
		},
		{
			expected: storage.SourceType_GO,
			env: &v4.Environment{
				PackageDb: "go:usr/local/bin/scanner",
			},
		},
		{
			expected: storage.SourceType_JAVA,
			env: &v4.Environment{
				PackageDb: "file:pkg.jar",
			},
		},
		{
			expected: storage.SourceType_JAVA,
			env: &v4.Environment{
				PackageDb: "jar:pkg.jar",
			},
		},
		{
			expected: storage.SourceType_JAVA,
			env: &v4.Environment{
				PackageDb: "maven:pkg.jar",
			},
		},
		{
			expected: storage.SourceType_NODEJS,
			env: &v4.Environment{
				PackageDb: "nodejs:package.json",
			},
		},
		{
			expected: storage.SourceType_PYTHON,
			env: &v4.Environment{
				PackageDb: "python:hello/.egg-info",
			},
		},
		{
			expected: storage.SourceType_RUBY,
			env: &v4.Environment{
				PackageDb: "ruby:opt/specifications/howdy.gemspec",
			},
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.expected.String(), func(t *testing.T) {
			source := sourceType(testcase.env)
			assert.Equal(t, testcase.expected, source)
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
