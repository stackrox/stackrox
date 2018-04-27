package risk

import (
	"fmt"
	"math"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

const (
	vulnsHeading      = "Image Vulnerabilities"
	saturationCeiling = 100
)

// vulnerabilitiesMultiplier is a scorer for the vulnerabilities in a deployment
type vulnerabilitiesMultiplier struct{}

// newVulnerabilitiesMultiplier scores the data based on the CVSS scores and number of CVEs
func newVulnerabilitiesMultiplier() *vulnerabilitiesMultiplier {
	return &vulnerabilitiesMultiplier{}
}

// Score takes a deployment and evaluates its risk based on vulnerabilties
func (c *vulnerabilitiesMultiplier) Score(deployment *v1.Deployment) *v1.Risk_Result {
	var cvssSum float32
	cvssMin := math.MaxFloat64
	cvssMax := -math.MaxFloat64
	var numCVEs int
	for _, container := range deployment.GetContainers() {
		for _, component := range container.GetImage().GetScan().GetComponents() {
			for _, vuln := range component.GetVulns() {
				// Sometimes if the vuln doesn't have a CVSS score then it is unknown and we'll exclude it during scoring
				if vuln.GetCvss() == 0 {
					continue
				}
				cvssMax = math.Max(float64(vuln.GetCvss()), cvssMax)
				cvssMin = math.Min(float64(vuln.GetCvss()), cvssMin)
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
		Name: vulnsHeading,
		Factors: []string{
			fmt.Sprintf("Image contains %d CVEs with CVSS scores ranging between %0.1f and %0.1f", numCVEs, cvssMin, cvssMax),
		},
		Score: score,
	}
}
