package scan

import (
	"errors"
	"fmt"
)

// ErrVulnerabilityFound occurs if an image scan reveals at least one vulnerability.
var ErrVulnerabilityFound = errors.New("vulnerabilities found")

// NewErrVulnerabilityFound creates an errVulnerabilityFound with the number of vulnerabilities
// in the explanation.
func NewErrVulnerabilityFound(num int) error {
	return fmt.Errorf("%w: %d vulnerabilities", ErrVulnerabilityFound, num)
}
