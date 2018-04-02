package risk

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

const (
	saturationCeiling = 100
)

// VulnerabilitiesMultiplier is a scorer for the vulnerabilities in a deployment
type VulnerabilitiesMultiplier struct{}

// NewVulnerabilitiesMultiplier scores the data based on the CVSS scores and number of CVEs
func NewVulnerabilitiesMultiplier() *VulnerabilitiesMultiplier {
	return &VulnerabilitiesMultiplier{}
}

// Score takes a deployment and evaluates its risk based on vulnerabilties
func (c *VulnerabilitiesMultiplier) Score(deployment *v1.Deployment) *v1.Risk_Result {
	var cvssSum float32
	var numCVEs int
	for _, container := range deployment.GetContainers() {
		for _, component := range container.GetImage().GetScan().GetComponents() {
			for _, vuln := range component.GetVulns() {
				// Sometimes if the vuln doesn't have a CVSS score then it is unknown and we'll exclude it during scoring
				if vuln.GetCvss() == 0 {
					continue
				}
				cvssSum += vuln.GetCvss() * vuln.GetCvss() / 10
				numCVEs++
			}
		}
	}
	// This does not contribute to the overall risk of the container
	if cvssSum == 0 {
		return nil
	} else if cvssSum > saturationCeiling {
		cvssSum = saturationCeiling
	}
	score := (cvssSum / saturationCeiling) + 1
	return &v1.Risk_Result{
		Name: "Vulnerability Heuristic",
		Factors: []string{
			fmt.Sprintf("Normalized and discounted sum of %d CVSS scores", numCVEs),
		},
		Score: score,
	}
}
