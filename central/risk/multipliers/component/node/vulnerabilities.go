package node

import "github.com/stackrox/rox/central/risk/multipliers/component"

const (
	// VulnerabilitiesHeading is the risk result name for scores calculated by this multiplier.
	VulnerabilitiesHeading = "Node Component Vulnerabilities"
)

// NewVulnerabilities provides a multiplier that scores the data based on the CVSS scores and number of CVEs
func NewVulnerabilities() component.Multiplier {
	return component.NewVulnerabilities("Node", VulnerabilitiesHeading)
}
