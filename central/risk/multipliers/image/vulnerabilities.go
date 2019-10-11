package image

import (
	"context"
	"fmt"
	"math"

	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/generated/storage"
)

const (
	// ImageVulnerabilitiesHeading is the risk result name for scores calculated by this multiplier.
	ImageVulnerabilitiesHeading = "Image Vulnerabilities"

	vulnSaturation = 100
	vulnMaxScore   = 4
)

// vulnerabilitiesMultiplier is a scorer for the vulnerabilities in an image
type vulnerabilitiesMultiplier struct{}

// NewVulnerabilities provides a multiplier that scores the data based on the CVSS scores and number of CVEs
func NewVulnerabilities() Multiplier {
	return &vulnerabilitiesMultiplier{}
}

// Score takes a image and evaluates its risk based on vulnerabilties
func (c *vulnerabilitiesMultiplier) Score(ctx context.Context, image *storage.Image) *storage.Risk_Result {
	var cvssSum float32
	var factors []*storage.Risk_Result_Factor
	cvssMin := math.MaxFloat64
	cvssMax := -math.MaxFloat64
	numCVEs := 0
	for _, component := range image.GetScan().GetComponents() {
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

	if numCVEs > 0 {
		factors = append(factors, &storage.Risk_Result_Factor{
			Message: fmt.Sprintf("Image %q contains %d CVEs with CVSS scores ranging between %0.1f and %0.1f",
				image.GetName().GetFullName(), numCVEs, cvssMin, cvssMax),
		})
	}

	// This does not contribute to the overall risk of the container
	if len(factors) == 0 {
		return nil
	}
	score := multipliers.NormalizeScore(cvssSum, vulnSaturation, vulnMaxScore)

	return &storage.Risk_Result{
		Name:    ImageVulnerabilitiesHeading,
		Factors: factors,
		Score:   score,
	}
}
