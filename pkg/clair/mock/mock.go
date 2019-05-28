package mock

import (
	clairV1 "github.com/coreos/clair/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// GetTestVulns returns test clair vulns and also the expected converted proto vulns
func GetTestVulns() ([]clairV1.Vulnerability, []*storage.Vulnerability) {
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
	}
	protoVulns := []*storage.Vulnerability{
		{
			Cve:  "CVE-2017-16231",
			Link: "https://security-tracker.debian.org/tracker/CVE-2017-16231",
			SetFixedBy: &storage.Vulnerability_FixedBy{
				FixedBy: "fixedby",
			},
		},
		{
			Cve:     "CVE-2017-7246",
			Link:    "https://security-tracker.debian.org/tracker/CVE-2017-7246",
			Summary: "Stack-based buffer overflow in the pcre32_copy_substring...",
			Cvss:    6.8,
			SetFixedBy: &storage.Vulnerability_FixedBy{
				FixedBy: "",
			},
			CvssV2: &storage.CVSSV2{
				Vector:           "AV:N/AC:L/Au:S/C:N/I:N",
				AttackVector:     storage.CVSSV2_ATTACK_NETWORK,
				AccessComplexity: storage.CVSSV2_ACCESS_LOW,
				Authentication:   storage.CVSSV2_AUTH_SINGLE,
				Confidentiality:  storage.CVSSV2_IMPACT_NONE,
				Integrity:        storage.CVSSV2_IMPACT_NONE,
				Availability:     storage.CVSSV2_IMPACT_NONE,
			},
		},
	}
	return quayVulns, protoVulns
}

// GetTestFeatures returns test clair features and also the expected converted proto components
func GetTestFeatures() ([]clairV1.Feature, []*storage.ImageScanComponent) {
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
	protoComponents := []*storage.ImageScanComponent{
		{
			Name:    "nginx-module-geoip",
			Version: "1.10.3-1~jessie",
			Vulns:   []*storage.Vulnerability{},
		},
		{
			Name:    "pcre3",
			Version: "2:8.35-3.3+deb8u4",
			Vulns:   protoVulns,
		},
	}
	return quayFeatures, protoComponents
}
