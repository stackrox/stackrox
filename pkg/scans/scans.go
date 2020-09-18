package scans

import "strings"

const (
	cveLinkPrefix       = "https://nvd.nist.gov/vuln/detail/"
	cveRedHatLinkPrefix = "https://access.redhat.com/security/cve/"
)

// GetVulnLink returns the default vulnerability link if the scanner does not provide one
func GetVulnLink(cve string) string {
	return cveLinkPrefix + cve
}

// GetRedHatVulnLink returns the default vulnerability link for rhel/centos images
func GetRedHatVulnLink(cve string) string {
	return cveRedHatLinkPrefix + strings.ToUpper(cve)
}
