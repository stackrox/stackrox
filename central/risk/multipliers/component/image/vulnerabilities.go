package image

import "github.com/stackrox/stackrox/central/risk/multipliers/component"

const (
	// VulnerabilitiesHeading is the risk result name for scores calculated by this multiplier.
	VulnerabilitiesHeading = "Image Component Vulnerabilities"
)

// NewVulnerabilities provides a multiplier that scores the data based on the CVSS scores and number of CVEs
func NewVulnerabilities() component.Multiplier {
	return component.NewVulnerabilities("Image", VulnerabilitiesHeading)
}
