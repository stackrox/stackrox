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
		storage.EmbeddedVulnerability_builder{
			Cve:     "CVE-2017-7246",
			Link:    "https://security-tracker.debian.org/tracker/CVE-2017-7246",
			Summary: "Stack-based buffer overflow in the pcre32_copy_substring function in pcre_get.c in libpcre1 in PCRE 8.40 allows remote attackers to cause a denial of service (WRITE of size 268) or possibly have unspecified other impact via a crafted file.",
			Cvss:    6.8,
			FixedBy: proto.String(""),
			CvssV2: storage.CVSSV2_builder{
				Vector:           "AV:N/AC:M/Au:N/C:P/I:P/A:P",
				Score:            6.8,
				AttackVector:     storage.CVSSV2_ATTACK_NETWORK,
				AccessComplexity: storage.CVSSV2_ACCESS_MEDIUM,
				Authentication:   storage.CVSSV2_AUTH_NONE,
				Confidentiality:  storage.CVSSV2_IMPACT_PARTIAL,
				Integrity:        storage.CVSSV2_IMPACT_PARTIAL,
				Availability:     storage.CVSSV2_IMPACT_PARTIAL,
				Severity:         storage.CVSSV2_MEDIUM,
			}.Build(),
			VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		}.Build(),
		storage.EmbeddedVulnerability_builder{
			Cve:          "CVE-2017-7247",
			Link:         "https://security-tracker.debian.org/tracker/CVE-2017-7247",
			Summary:      "Stack-based buffer overflow in the pcre32_copy_substring function in pcre_get.c in libpcre1 in PCRE 8.40 allows remote attackers to cause a denial of service (WRITE of size 268) or possibly have unspecified other impact via a crafted file.",
			Cvss:         7.5,
			FixedBy:      proto.String(""),
			ScoreVersion: storage.EmbeddedVulnerability_V3,
			CvssV2: storage.CVSSV2_builder{
				Vector:              "AV:N/AC:L/Au:S/C:N/I:N/A:C",
				ExploitabilityScore: 8.0,
				ImpactScore:         6.9,
				Score:               6.8,
				AttackVector:        storage.CVSSV2_ATTACK_NETWORK,
				AccessComplexity:    storage.CVSSV2_ACCESS_LOW,
				Authentication:      storage.CVSSV2_AUTH_SINGLE,
				Confidentiality:     storage.CVSSV2_IMPACT_NONE,
				Integrity:           storage.CVSSV2_IMPACT_NONE,
				Availability:        storage.CVSSV2_IMPACT_COMPLETE,
				Severity:            storage.CVSSV2_MEDIUM,
			}.Build(),
			CvssV3: storage.CVSSV3_builder{
				Vector:              "CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:N/A:N",
				Score:               7.5,
				ExploitabilityScore: 3.9,
				ImpactScore:         3.6,
				AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
				AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
				PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_NONE,
				UserInteraction:     storage.CVSSV3_UI_NONE,
				Scope:               storage.CVSSV3_UNCHANGED,
				Confidentiality:     storage.CVSSV3_IMPACT_HIGH,
				Integrity:           storage.CVSSV3_IMPACT_NONE,
				Availability:        storage.CVSSV3_IMPACT_NONE,
				Severity:            storage.CVSSV3_HIGH,
			}.Build(),
			VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		}.Build(),
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
	eisc := &storage.EmbeddedImageScanComponent{}
	eisc.SetName("nginx-module-geoip")
	eisc.SetVersion("1.10.3-1~jessie")
	eisc.SetVulns([]*storage.EmbeddedVulnerability{})
	eisc.SetExecutables([]*storage.EmbeddedImageScanComponent_Executable{})
	eisc2 := &storage.EmbeddedImageScanComponent{}
	eisc2.SetName("pcre3")
	eisc2.SetVersion("2:8.35-3.3+deb8u4")
	eisc2.SetVulns(protoVulns[1:]) // cut out the nil value
	eisc2.SetExecutables([]*storage.EmbeddedImageScanComponent_Executable{})
	protoComponents := []*storage.EmbeddedImageScanComponent{
		eisc,
		eisc2,
	}
	return quayFeatures, protoComponents
}
