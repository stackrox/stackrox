package scan

import (
	"github.com/pkg/errors"
)

// errVulnerabilityFound occurs if an image scan reveals at least one vulnerability.
var errVulnerabilityFound = errors.New("vulnerabilities found")

// newErrVulnerabilityFound creates an errVulnerabilityFound with the number of vulnerabilities
// in the explanation.
func newErrVulnerabilityFound(num int) error {
	return errors.WithMessagef(errVulnerabilityFound, "found %d vulnerabilities", num)
}
