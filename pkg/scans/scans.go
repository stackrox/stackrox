package scans

const (
	cveLinkPrefix = "https://nvd.nist.gov/vuln/detail/"
)

// GetVulnLink returns the default vulnerability link if the scanner does not provide one
func GetVulnLink(cve string) string {
	return cveLinkPrefix + cve
}
