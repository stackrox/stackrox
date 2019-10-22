package image

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/central/risk/scorer/vulns"
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
	min, max, sum, num := vulns.ProcessComponents(image.GetScan().GetComponents())
	if num == 0 {
		return nil
	}

	score := multipliers.NormalizeScore(sum, vulnSaturation, vulnMaxScore)
	return &storage.Risk_Result{
		Name: ImageVulnerabilitiesHeading,
		Factors: []*storage.Risk_Result_Factor{
			{
				Message: fmt.Sprintf("Image %q contains %d CVEs with CVSS scores ranging between %0.1f and %0.1f",
					image.GetName().GetFullName(), num, min, max),
			},
		},
		Score: score,
	}
}
