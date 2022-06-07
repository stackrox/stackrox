package node

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/central/risk/scorer/vulns"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scancomponent"
)

const (
	// VulnerabilitiesHeading is the risk result name for scores calculated by this multiplier.
	VulnerabilitiesHeading = "Node Vulnerabilities"

	vulnSaturation = 100
	vulnMaxScore   = 4
)

// vulnerabilitiesMultiplier is a scorer for the vulnerabilities in a node
type vulnerabilitiesMultiplier struct{}

// NewVulnerabilities provides a multiplier that scores the data based on the CVSS scores and number of CVEs
func NewVulnerabilities() Multiplier {
	return &vulnerabilitiesMultiplier{}
}

// Score takes a image and evaluates its risk based on vulnerabilities
func (c *vulnerabilitiesMultiplier) Score(_ context.Context, node *storage.Node) *storage.Risk_Result {
	nodeComponents := node.GetScan().GetComponents()
	components := make([]scancomponent.ScanComponent, 0, len(nodeComponents))
	for _, nodeComponent := range nodeComponents {
		components = append(components, scancomponent.NewFromNodeComponent(nodeComponent))
	}
	min, max, sum, num := vulns.ProcessComponents(components)
	if num == 0 {
		return nil
	}

	score := multipliers.NormalizeScore(sum, vulnSaturation, vulnMaxScore)
	return &storage.Risk_Result{
		Name: VulnerabilitiesHeading,
		Factors: []*storage.Risk_Result_Factor{
			{
				Message: fmt.Sprintf("Node %q contains %d CVEs with severities ranging between %s and %s",
					node.GetName(), num, min.Severity, max.Severity),
			},
		},
		Score: score,
	}
}
