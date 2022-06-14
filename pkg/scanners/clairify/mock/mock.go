package mock

import (
	"github.com/stackrox/stackrox/generated/storage"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
)

// GetTestScannerVulns returns test clair vulns and also the expected converted proto vulns
func GetTestScannerVulns() ([]scannerV1.Vulnerability, []*storage.EmbeddedVulnerability) {
	m1 := scannerV1.Metadata{
		CvssV2: &scannerV1.CVSSMetadata{
			Score:               4.6,
			Vector:              "AV:L/AC:L/Au:N/C:P/I:P/A:P",
			ExploitabilityScore: 3.9,
			ImpactScore:         6.4,
		},
		CvssV3: &scannerV1.CVSSMetadata{
			Score:               4.9,
			Vector:              "CVSS:3.0/AV:L/AC:H/PR:N/UI:N/S:U/C:L/I:L/A:L",
			ExploitabilityScore: 1.4,
			ImpactScore:         3.4,
		},
	}
	m2 := scannerV1.Metadata{
		CvssV3: &scannerV1.CVSSMetadata{
			Score:               4.9,
			Vector:              "CVSS:3.0/AV:L/AC:H/PR:N/UI:N/S:U/C:L/I:L/A:L",
			ExploitabilityScore: 1.4,
			ImpactScore:         3.4,
		},
	}

	scannerVulns := []scannerV1.Vulnerability{
		{
			Link:        "https://security-tracker.debian.org/tracker/CVE-2017-7246",
			Description: "Stack-based buffer overflow in the pcre32_copy_substring function in pcre_get.c in libpcre1 in PCRE 8.40 allows remote attackers to cause a denial of service (WRITE of size 268) or possibly have unspecified other impact via a crafted file.",
			Name:        "CVE-2017-7246",
			MetadataV2:  &m1,
			FixedBy:     "fixedby",
		},
		{
			Link:        "https://security-tracker.debian.org/tracker/CVE-2017-7246",
			Description: "Stack-based buffer overflow in the pcre32_copy_substring function in pcre_get.c in libpcre1 in PCRE 8.40 allows remote attackers to cause a denial of service (WRITE of size 268) or possibly have unspecified other impact via a crafted file.",
			Name:        "CVE-2017-7247",
			MetadataV2:  &m2,
		},
	}

	protoVulns := []*storage.EmbeddedVulnerability{
		{
			Cve:     "CVE-2017-7246",
			Link:    "https://security-tracker.debian.org/tracker/CVE-2017-7246",
			Summary: "Stack-based buffer overflow in the pcre32_copy_substring function in pcre_get.c in libpcre1 in PCRE 8.40 allows remote attackers to cause a denial of service (WRITE of size 268) or possibly have unspecified other impact via a crafted file.",
			Cvss:    4.9,
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
				FixedBy: "fixedby",
			},
			ScoreVersion: storage.EmbeddedVulnerability_V3,
			CvssV2: &storage.CVSSV2{
				Vector:              "AV:L/AC:L/Au:N/C:P/I:P/A:P",
				Score:               4.6,
				ExploitabilityScore: 3.9,
				ImpactScore:         6.4,
				AttackVector:        storage.CVSSV2_ATTACK_LOCAL,
				AccessComplexity:    storage.CVSSV2_ACCESS_LOW,
				Authentication:      storage.CVSSV2_AUTH_NONE,
				Confidentiality:     storage.CVSSV2_IMPACT_PARTIAL,
				Integrity:           storage.CVSSV2_IMPACT_PARTIAL,
				Availability:        storage.CVSSV2_IMPACT_PARTIAL,
				Severity:            storage.CVSSV2_MEDIUM,
			},
			CvssV3: &storage.CVSSV3{
				Vector:              "CVSS:3.0/AV:L/AC:H/PR:N/UI:N/S:U/C:L/I:L/A:L",
				Score:               4.9,
				ExploitabilityScore: 1.4,
				ImpactScore:         3.4,
				AttackVector:        storage.CVSSV3_ATTACK_LOCAL,
				AttackComplexity:    storage.CVSSV3_COMPLEXITY_HIGH,
				PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_NONE,
				UserInteraction:     storage.CVSSV3_UI_NONE,
				Scope:               storage.CVSSV3_UNCHANGED,
				Confidentiality:     storage.CVSSV3_IMPACT_LOW,
				Integrity:           storage.CVSSV3_IMPACT_LOW,
				Availability:        storage.CVSSV3_IMPACT_LOW,
				Severity:            storage.CVSSV3_MEDIUM,
			},
			VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
		},
		{
			Cve:     "CVE-2017-7247",
			Link:    "https://security-tracker.debian.org/tracker/CVE-2017-7246",
			Summary: "Stack-based buffer overflow in the pcre32_copy_substring function in pcre_get.c in libpcre1 in PCRE 8.40 allows remote attackers to cause a denial of service (WRITE of size 268) or possibly have unspecified other impact via a crafted file.",
			Cvss:    4.9,
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
				FixedBy: "",
			},
			ScoreVersion: storage.EmbeddedVulnerability_V3,
			CvssV3: &storage.CVSSV3{
				Vector:              "CVSS:3.0/AV:L/AC:H/PR:N/UI:N/S:U/C:L/I:L/A:L",
				Score:               4.9,
				ExploitabilityScore: 1.4,
				ImpactScore:         3.4,
				AttackVector:        storage.CVSSV3_ATTACK_LOCAL,
				AttackComplexity:    storage.CVSSV3_COMPLEXITY_HIGH,
				PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_NONE,
				UserInteraction:     storage.CVSSV3_UI_NONE,
				Scope:               storage.CVSSV3_UNCHANGED,
				Confidentiality:     storage.CVSSV3_IMPACT_LOW,
				Integrity:           storage.CVSSV3_IMPACT_LOW,
				Availability:        storage.CVSSV3_IMPACT_LOW,
				Severity:            storage.CVSSV3_MEDIUM,
			},
			VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
		},
	}

	return scannerVulns, protoVulns
}
