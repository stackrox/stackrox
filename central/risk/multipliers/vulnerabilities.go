package multipliers

import (
	"fmt"
	"math"

	"github.com/stackrox/rox/generated/storage"
)

const (
	// VulnsHeading is the risk result name for scores calculated by this multiplier.
	VulnsHeading = "Image Vulnerabilities"

	vulnSaturation = 100
	vulnMaxScore   = 4
)

// vulnerabilitiesMultiplier is a scorer for the vulnerabilities in a deployment
type vulnerabilitiesMultiplier struct{}

// NewVulnerabilities provides a multiplier that scores the data based on the CVSS scores and number of CVEs
func NewVulnerabilities() Multiplier {
	return &vulnerabilitiesMultiplier{}
}

// Score takes a deployment and evaluates its risk based on vulnerabilties
func (c *vulnerabilitiesMultiplier) Score(deployment *storage.Deployment) *storage.Risk_Result {
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
	}
	score := normalizeScore(cvssSum, vulnSaturation, vulnMaxScore)
	return &storage.Risk_Result{
		Name: VulnsHeading,
		Factors: []*storage.Risk_Result_Factor{
			{Message: fmt.Sprintf("Image contains %d CVEs with CVSS scores ranging between %0.1f and %0.1f", numCVEs, cvssMin, cvssMax)},
		},
		Score: score,
	}
}
