package mock

import (
	"github.com/stackrox/rox/generated/storage"
	clairV1 "github.com/stackrox/scanner/api/v1"
	"google.golang.org/protobuf/proto"
)

// GetTestVulns returns test clair vulns and also the expected converted proto vulns
func GetTestVulns() ([]clairV1.Vulnerability, []*storage.EmbeddedVulnerability) {
	quayVulns := []clairV1.Vulnerability{
		{
			Link:    "https://security-tracker.debian.org/tracker/CVE-2017-16231",
			Name:    "CVE-2017-16231",
			FixedBy: "fixedby",
		},
		{
			Link:        "https://security-tracker.debian.org/tracker/CVE-2017-7246",
			Description: "Stack-based buffer overflow in the pcre32_copy_substring function in pcre_get.c in libpcre1 in PCRE 8.40 allows remote attackers to cause a denial of service (WRITE of size 268) or possibly have unspecified other impact via a crafted file.",
			Name:        "CVE-2017-7246",
			Metadata: map[string]interface{}{
				"NVD": map[string]interface{}{
					"CVSSv2": map[string]interface{}{
						"Score":   6.8,
						"Vectors": "AV:N/AC:M/Au:N/C:P/I:P/A:P",
					},
				},
			},
		},
		{
			Link:        "https://security-tracker.debian.org/tracker/CVE-2017-7247",
			Description: "Stack-based buffer overflow in the pcre32_copy_substring function in pcre_get.c in libpcre1 in PCRE 8.40 allows remote attackers to cause a denial of service (WRITE of size 268) or possibly have unspecified other impact via a crafted file.",
			Name:        "CVE-2017-7247",
			Metadata: map[string]interface{}{
				"NVD": map[string]interface{}{
					"CVSSv2": map[string]interface{}{
						"Score":               6.8,
						"Vectors":             "AV:N/AC:L/Au:S/C:N/I:N/A:C",
						"ExploitabilityScore": 8.0,
						"ImpactScore":         6.9,
					},
					"CVSSv3": map[string]interface{}{
						"Vectors":             "CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:N/A:N",
						"Score":               7.5,
						"ExploitabilityScore": 3.9,
						"ImpactScore":         3.6,
					},
				},
			},
		},
	}
	protoVulns := []*storage.EmbeddedVulnerability{
		nil,
		func() *storage.EmbeddedVulnerability {
			fixedBy := ""
			attackVector := storage.CVSSV2_ATTACK_NETWORK
			accessComplexity := storage.CVSSV2_ACCESS_MEDIUM
			authentication := storage.CVSSV2_AUTH_NONE
			confidentiality := storage.CVSSV2_IMPACT_PARTIAL
			integrity := storage.CVSSV2_IMPACT_PARTIAL
			availability := storage.CVSSV2_IMPACT_PARTIAL
			severity := storage.CVSSV2_MEDIUM
			vulnType := storage.EmbeddedVulnerability_IMAGE_VULNERABILITY

			return storage.EmbeddedVulnerability_builder{
				Cve:     proto.String("CVE-2017-7246"),
				Link:    proto.String("https://security-tracker.debian.org/tracker/CVE-2017-7246"),
				Summary: proto.String("Stack-based buffer overflow in the pcre32_copy_substring function in pcre_get.c in libpcre1 in PCRE 8.40 allows remote attackers to cause a denial of service (WRITE of size 268) or possibly have unspecified other impact via a crafted file."),
				Cvss:    proto.Float32(6.8),
				FixedBy: &fixedBy,
				CvssV2: storage.CVSSV2_builder{
					Vector:           proto.String("AV:N/AC:M/Au:N/C:P/I:P/A:P"),
					Score:            proto.Float32(6.8),
					AttackVector:     &attackVector,
					AccessComplexity: &accessComplexity,
					Authentication:   &authentication,
					Confidentiality:  &confidentiality,
					Integrity:        &integrity,
					Availability:     &availability,
					Severity:         &severity,
				}.Build(),
				VulnerabilityType: &vulnType,
			}.Build()
		}(),
		func() *storage.EmbeddedVulnerability {
			fixedBy := ""
			scoreVersion := storage.EmbeddedVulnerability_V3
			vulnType := storage.EmbeddedVulnerability_IMAGE_VULNERABILITY

			// CVSS V2 vars
			v2AttackVector := storage.CVSSV2_ATTACK_NETWORK
			v2AccessComplexity := storage.CVSSV2_ACCESS_LOW
			v2Authentication := storage.CVSSV2_AUTH_SINGLE
			v2Confidentiality := storage.CVSSV2_IMPACT_NONE
			v2Integrity := storage.CVSSV2_IMPACT_NONE
			v2Availability := storage.CVSSV2_IMPACT_COMPLETE
			v2Severity := storage.CVSSV2_MEDIUM

			// CVSS V3 vars
			v3AttackVector := storage.CVSSV3_ATTACK_NETWORK
			v3AttackComplexity := storage.CVSSV3_COMPLEXITY_LOW
			v3PrivilegesRequired := storage.CVSSV3_PRIVILEGE_NONE
			v3UserInteraction := storage.CVSSV3_UI_NONE
			v3Scope := storage.CVSSV3_UNCHANGED
			v3Confidentiality := storage.CVSSV3_IMPACT_HIGH
			v3Integrity := storage.CVSSV3_IMPACT_NONE
			v3Availability := storage.CVSSV3_IMPACT_NONE
			v3Severity := storage.CVSSV3_HIGH

			return storage.EmbeddedVulnerability_builder{
				Cve:     proto.String("CVE-2017-7247"),
				Link:    proto.String("https://security-tracker.debian.org/tracker/CVE-2017-7247"),
				Summary: proto.String("Stack-based buffer overflow in the pcre32_copy_substring function in pcre_get.c in libpcre1 in PCRE 8.40 allows remote attackers to cause a denial of service (WRITE of size 268) or possibly have unspecified other impact via a crafted file."),
				Cvss:    proto.Float32(7.5),
				FixedBy: &fixedBy,
				ScoreVersion: &scoreVersion,
				CvssV2: storage.CVSSV2_builder{
					Vector:              proto.String("AV:N/AC:L/Au:S/C:N/I:N/A:C"),
					ExploitabilityScore: proto.Float32(8.0),
					ImpactScore:         proto.Float32(6.9),
					Score:               proto.Float32(6.8),
					AttackVector:        &v2AttackVector,
					AccessComplexity:    &v2AccessComplexity,
					Authentication:      &v2Authentication,
					Confidentiality:     &v2Confidentiality,
					Integrity:           &v2Integrity,
					Availability:        &v2Availability,
					Severity:            &v2Severity,
				}.Build(),
				CvssV3: storage.CVSSV3_builder{
					Vector:              proto.String("CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:N/A:N"),
					Score:               proto.Float32(7.5),
					ExploitabilityScore: proto.Float32(3.9),
					ImpactScore:         proto.Float32(3.6),
					AttackVector:        &v3AttackVector,
					AttackComplexity:    &v3AttackComplexity,
					PrivilegesRequired:  &v3PrivilegesRequired,
					UserInteraction:     &v3UserInteraction,
					Scope:               &v3Scope,
					Confidentiality:     &v3Confidentiality,
					Integrity:           &v3Integrity,
					Availability:        &v3Availability,
					Severity:            &v3Severity,
				}.Build(),
				VulnerabilityType: &vulnType,
			}.Build()
		}(),
	}
	return quayVulns, protoVulns
}

// GetTestFeatures returns test clair features and also the expected converted proto components
func GetTestFeatures() ([]clairV1.Feature, []*storage.EmbeddedImageScanComponent) {
	quayVulns, protoVulns := GetTestVulns()
	quayFeatures := []clairV1.Feature{
		{
			Name:    "nginx-module-geoip",
			Version: "1.10.3-1~jessie",
		},
		{
			Name:            "pcre3",
			Version:         "2:8.35-3.3+deb8u4",
			Vulnerabilities: quayVulns,
		},
	}
	protoComponents := []*storage.EmbeddedImageScanComponent{
		storage.EmbeddedImageScanComponent_builder{
			Name:        proto.String("nginx-module-geoip"),
			Version:     proto.String("1.10.3-1~jessie"),
			Vulns:       []*storage.EmbeddedVulnerability{},
			Executables: []*storage.EmbeddedImageScanComponent_Executable{},
		}.Build(),
		storage.EmbeddedImageScanComponent_builder{
			Name:        proto.String("pcre3"),
			Version:     proto.String("2:8.35-3.3+deb8u4"),
			Vulns:       protoVulns[1:], // cut out the nil value
			Executables: []*storage.EmbeddedImageScanComponent_Executable{},
		}.Build(),
	}
	return quayFeatures, protoComponents
}
