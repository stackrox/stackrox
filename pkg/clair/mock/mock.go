package mock

import (
	"github.com/stackrox/rox/generated/storage"
	clairV1 "github.com/stackrox/scanner/api/v1"
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
						"Vectors": "AV:N/AC:L/Au:S/C:N/I:N",
					},
				},
			},
		},
		{
			Link:        "https://security-tracker.debian.org/tracker/CVE-2017-7246",
			Description: "Stack-based buffer overflow in the pcre32_copy_substring function in pcre_get.c in libpcre1 in PCRE 8.40 allows remote attackers to cause a denial of service (WRITE of size 268) or possibly have unspecified other impact via a crafted file.",
			Name:        "CVE-2017-7247",
			Metadata: map[string]interface{}{
				"NVD": map[string]interface{}{
					"CVSSv2": map[string]interface{}{
						"Score":               7,
						"Vectors":             "AV:N/AC:L/Au:S/C:N/I:N",
						"ExploitabilityScore": 10,
						"ImpactScore":         2.9,
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
		{
			Cve:     "CVE-2017-7246",
			Link:    "https://security-tracker.debian.org/tracker/CVE-2017-7246",
			Summary: "Stack-based buffer overflow in the pcre32_copy_substring function in pcre_get.c in libpcre1 in PCRE 8.40 allows remote attackers to cause a denial of service (WRITE of size 268) or possibly have unspecified other impact via a crafted file.",
			Cvss:    6.8,
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
				FixedBy: "",
			},
			CvssV2: &storage.CVSSV2{
				Vector:           "AV:N/AC:L/Au:S/C:N/I:N",
				Score:            6.8,
				AttackVector:     storage.CVSSV2_ATTACK_NETWORK,
				AccessComplexity: storage.CVSSV2_ACCESS_LOW,
				Authentication:   storage.CVSSV2_AUTH_SINGLE,
				Confidentiality:  storage.CVSSV2_IMPACT_NONE,
				Integrity:        storage.CVSSV2_IMPACT_NONE,
				Availability:     storage.CVSSV2_IMPACT_NONE,
				Severity:         storage.CVSSV2_MEDIUM,
			},
			VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		},
		{
			Cve:     "CVE-2017-7247",
			Link:    "https://security-tracker.debian.org/tracker/CVE-2017-7246",
			Summary: "Stack-based buffer overflow in the pcre32_copy_substring function in pcre_get.c in libpcre1 in PCRE 8.40 allows remote attackers to cause a denial of service (WRITE of size 268) or possibly have unspecified other impact via a crafted file.",
			Cvss:    7.5,
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
				FixedBy: "",
			},
			ScoreVersion: storage.EmbeddedVulnerability_V3,
			CvssV2: &storage.CVSSV2{
				Vector:              "AV:N/AC:L/Au:S/C:N/I:N",
				ExploitabilityScore: 10,
				ImpactScore:         2.9,
				Score:               7,
				AttackVector:        storage.CVSSV2_ATTACK_NETWORK,
				AccessComplexity:    storage.CVSSV2_ACCESS_LOW,
				Authentication:      storage.CVSSV2_AUTH_SINGLE,
				Confidentiality:     storage.CVSSV2_IMPACT_NONE,
				Integrity:           storage.CVSSV2_IMPACT_NONE,
				Availability:        storage.CVSSV2_IMPACT_NONE,
				Severity:            storage.CVSSV2_HIGH,
			},
			CvssV3: &storage.CVSSV3{
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
			},
			VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		},
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
		{
			Name:        "nginx-module-geoip",
			Version:     "1.10.3-1~jessie",
			Vulns:       []*storage.EmbeddedVulnerability{},
			Executables: []*storage.EmbeddedImageScanComponent_Executable{},
		},
		{
			Name:        "pcre3",
			Version:     "2:8.35-3.3+deb8u4",
			Vulns:       protoVulns[1:], // cut out the nil value
			Executables: []*storage.EmbeddedImageScanComponent_Executable{},
		},
	}
	return quayFeatures, protoComponents
}
