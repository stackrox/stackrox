package image

import (
	"context"
	"fmt"
	"strings"

	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/central/risk/scorer/vulns"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/scancomponent"
)

const (
	// VulnerabilitiesHeading is the risk result name for scores calculated by this multiplier.
	VulnerabilitiesHeading = "Image Vulnerabilities"

	vulnSaturation = 100
	vulnMaxScore   = 4
)

// vulnerabilitiesMultiplier is a scorer for the vulnerabilities in an image
type vulnerabilitiesMultiplier struct{}

// NewVulnerabilities provides a multiplier that scores the data based on the CVSS scores and number of CVEs
func NewVulnerabilities() Multiplier {
	return &vulnerabilitiesMultiplier{}
}

// Score takes an image and evaluates its risk based on vulnerabilities
func (c *vulnerabilitiesMultiplier) Score(_ context.Context, image *storage.Image) *storage.Risk_Result {
	log := logging.LoggerForModule()
	if strings.Contains(image.GetName().GetFullName(), "log4j") || strings.Contains(image.GetName().GetFullName(), "nginx") {
		log.Errorf("[Score - Images] STARTING scoring for image %s", image.GetName().GetFullName())
	}

	imgComponents := image.GetScan().GetComponents()
	components := make([]scancomponent.ScanComponent, 0, len(imgComponents))
	for _, imgComponent := range imgComponents {
		components = append(components, imgComponent)
	}
	min, max, sum, num := vulns.ProcessComponents(components)
	if num == 0 {
		return nil
	}


	if strings.Contains(image.GetName().GetFullName(), "log4j") || strings.Contains(image.GetName().GetFullName(), "nginx") {
		log.Errorf("[Score - Images] END scoring for image %s and got %v vulns", image.GetName().GetFullName(), num)
	}

	score := multipliers.NormalizeScore(sum, vulnSaturation, vulnMaxScore)
	return &storage.Risk_Result{
		Name: VulnerabilitiesHeading,
		Factors: []*storage.Risk_Result_Factor{
			{
				Message: fmt.Sprintf("Image %q contains %d CVEs with severities ranging between %s and %s",
					image.GetName().GetFullName(), num, min.Severity, max.Severity),
			},
		},
		Score: score,
	}
}
