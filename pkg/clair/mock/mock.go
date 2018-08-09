package mock

import (
	clairV1 "github.com/coreos/clair/api/v1"
	"github.com/stackrox/rox/generated/api/v1"
)

// GetTestVulns returns test clair vulns and also the expected converted proto vulns
func GetTestVulns() ([]clairV1.Vulnerability, []*v1.Vulnerability) {
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
						"Score": 6.8,
					},
				},
			},
		},
	}
	protoVulns := []*v1.Vulnerability{
		{
			Cve:  "CVE-2017-16231",
			Link: "https://security-tracker.debian.org/tracker/CVE-2017-16231",
			SetFixedBy: &v1.Vulnerability_FixedBy{
				FixedBy: "fixedby",
			},
		},
		{
			Cve:     "CVE-2017-7246",
			Link:    "https://security-tracker.debian.org/tracker/CVE-2017-7246",
			Summary: "Stack-based buffer overflow in the pcre32_copy_substring function in pcre_get.c in libpcre1 in PCRE 8.40 allows remote attackers to cause a denial of service (WRITE of size 268) or possibly have unspecified other impact via a crafted file.",
			Cvss:    6.8,
			SetFixedBy: &v1.Vulnerability_FixedBy{
				FixedBy: "",
			},
		},
	}
	return quayVulns, protoVulns
}

// GetTestFeatures returns test clair features and also the expected converted proto components
func GetTestFeatures() ([]clairV1.Feature, []*v1.ImageScanComponent) {
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
	protoComponents := []*v1.ImageScanComponent{
		{
			Name:    "nginx-module-geoip",
			Version: "1.10.3-1~jessie",
			Vulns:   []*v1.Vulnerability{},
		},
		{
			Name:    "pcre3",
			Version: "2:8.35-3.3+deb8u4",
			Vulns:   protoVulns,
		},
	}
	return quayFeatures, protoComponents
}
