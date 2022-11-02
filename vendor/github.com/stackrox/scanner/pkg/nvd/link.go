package nvd

import "fmt"

// Link returns a CVE link to NVD
func Link(cve string) string {
	return fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", cve)
}
