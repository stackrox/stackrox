package fixtures

import (
	v1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stackrox/scanner/pkg/component"
)

// ScannerFeaturesV1 returns a slice of *v1.Feature.
func ScannerFeaturesV1() []*v1.Feature {
	return []*v1.Feature{
		{
			Name:         "rpm",
			Version:      "4.16.0",
			AddedByLayer: "sha256:idk0",
			ProvidedExecutables: []*v1.Executable{
				{
					Path: "/bin/rpm",
					RequiredFeatures: []*v1.FeatureNameVersion{
						{
							Name:    "glibc",
							Version: "1",
						},
						{
							Name:    "lib.so",
							Version: "2",
						},
					},
				},
			},
			Vulnerabilities: []*v1.Vulnerability{
				{
					Name:        "CVE-2022-1234",
					Description: "This is the worst vulnerability I have ever seen",
					Link:        "https://access.redhat.com/security/cve/CVE-2022-1234",
					Severity:    "Important",
					FixedBy:     "4.16.1",
					MetadataV2: &v1.Metadata{
						CvssV2: &v1.CVSSMetadata{
							Vector:              "AV:A/AC:M/Au:M/C:N/I:P/A:C",
							Score:               5.4,
							ExploitabilityScore: 3.5,
							ImpactScore:         7.8,
						},
						CvssV3: &v1.CVSSMetadata{
							Vector:              "CVSS:3.1/AV:A/AC:L/PR:L/UI:N/S:U/C:L/I:N/A:H",
							Score:               6.3,
							ExploitabilityScore: 2.1,
							ImpactScore:         4.2,
						},
					},
				},
				{
					Name:        "CVE-2022-1235",
					Description: "This is the second worst vulnerability I have ever seen",
					Link:        "https://access.redhat.com/security/cve/CVE-2022-1235",
					Severity:    "Moderate",
					MetadataV2: &v1.Metadata{
						CvssV2: &v1.CVSSMetadata{
							Vector:              "AV:A/AC:M/Au:M/C:N/I:P/A:C",
							Score:               5.4,
							ExploitabilityScore: 3.5,
							ImpactScore:         7.8,
						},
					},
				},
			},
			FixedBy: "4.16.1",
		},
		{
			Name:         "curl",
			Version:      "1",
			AddedByLayer: "sha256:idk0",
		},
		{
			Name:         "java.jar",
			Version:      "1",
			FeatureType:  component.JavaSourceType.String(),
			Location:     "/java/jar/path/java.jar",
			AddedByLayer: "sha256:idk1",
		},
	}
}
