package image

import (
	"context"
	"fmt"
	"math"

	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/central/risk/scorer/vulns"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stackrox/rox/pkg/set"
)

const (
	// VulnerabilitiesHeading is the risk result name for scores calculated by this multiplier.
	VulnerabilitiesHeading = "Image Vulnerabilities"

	vulnSaturation = 100
	vulnMaxScore   = 4

	lowVulnContribution          = 0.05
	lowVulnContributionCap       = 5
	moderateVulnContribution     = 1
	moderateVulnContributionCap  = 20
	importantVulnContribution    = 8
	importantVulnContributionCap = 40
	criticalVulnContribution     = 25
	criticalVulnContributionCap  = 75

	networkAttackVectorBonus = 5
	kevOrRansomwareBonus     = 50
	highEPSSBonus            = 25
	highEPSSPercentile       = 0.95
)

// vulnerabilitiesMultiplier is a scorer for the vulnerabilities in an image
type vulnerabilitiesMultiplier struct{}

// NewVulnerabilities provides a multiplier that scores known-severity vulnerability findings across image components.
func NewVulnerabilities() Multiplier {
	return &vulnerabilitiesMultiplier{}
}

// Score takes an image and evaluates its risk based on vulnerabilities
func (c *vulnerabilitiesMultiplier) Score(_ context.Context, image *storage.Image) *storage.Risk_Result {
	imgComponents := image.GetScan().GetComponents()
	min, max, score, num := processImageVulnerabilities(imgComponents)
	if num == 0 {
		return nil
	}

	return &storage.Risk_Result{
		Name: VulnerabilitiesHeading,
		Factors: []*storage.Risk_Result_Factor{
			{
				Message: fmt.Sprintf("Image %q contains %d distinct CVEs with severities ranging between %s and %s",
					image.GetName().GetFullName(), countDistinctCVEs(imgComponents), min.Severity, max.Severity),
			},
		},
		Score: multipliers.NormalizeScore(score, vulnSaturation, vulnMaxScore),
	}
}

// ScoreV2 takes an image and evaluates its risk based on vulnerabilities
func (c *vulnerabilitiesMultiplier) ScoreV2(_ context.Context, image *storage.ImageV2) *storage.Risk_Result {
	imgComponents := image.GetScan().GetComponents()
	min, max, score, num := processImageVulnerabilities(imgComponents)
	if num == 0 {
		return nil
	}

	return &storage.Risk_Result{
		Name: VulnerabilitiesHeading,
		Factors: []*storage.Risk_Result_Factor{
			{
				Message: fmt.Sprintf("Image %q contains %d distinct CVEs with severities ranging between %s and %s",
					image.GetName().GetFullName(), countDistinctCVEs(imgComponents), min.Severity, max.Severity),
			},
		},
		Score: multipliers.NormalizeScore(score, vulnSaturation, vulnMaxScore),
	}
}

func processImageVulnerabilities(imgComponents []*storage.EmbeddedImageScanComponent) (min, max vulns.ComponentScore, score float32, num int) {
	components := make([]scancomponent.ScanComponent, 0, len(imgComponents))
	for _, imgComponent := range imgComponents {
		components = append(components, scancomponent.NewFromImageComponent(imgComponent))
	}
	min, max, _, num = vulns.ProcessComponents(components)
	if num == 0 {
		return vulns.ComponentScore{}, vulns.ComponentScore{}, 0, 0
	}

	var lowScore, moderateScore, importantScore, criticalScore float32
	for _, component := range imgComponents {
		for _, vuln := range component.GetVulns() {
			switch vuln.GetSeverity() {
			case storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY:
				lowScore += vulnerabilityContribution(vuln, lowVulnContribution)
			case storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY:
				moderateScore += vulnerabilityContribution(vuln, moderateVulnContribution)
			case storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY:
				importantScore += vulnerabilityContribution(vuln, importantVulnContribution)
			case storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY:
				criticalScore += vulnerabilityContribution(vuln, criticalVulnContribution)
			}
		}
	}

	score = minFloat32(lowScore, lowVulnContributionCap) +
		minFloat32(moderateScore, moderateVulnContributionCap) +
		minFloat32(importantScore, importantVulnContributionCap) +
		minFloat32(criticalScore, criticalVulnContributionCap)
	return min, max, score, num
}

func vulnerabilityContribution(vuln *storage.EmbeddedVulnerability, base float32) float32 {
	contribution := base
	if hasNetworkAttackVector(vuln) && vuln.GetSeverity() >= storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY {
		contribution += networkAttackVectorBonus
	}
	if vuln.GetCisaKev() || vuln.GetExploit().GetKnownRansomwareCampaignUse() != "" {
		contribution += kevOrRansomwareBonus
	}
	if vuln.GetEpss().GetEpssPercentile() >= highEPSSPercentile {
		contribution += highEPSSBonus
	}
	return contribution
}

func hasNetworkAttackVector(vuln *storage.EmbeddedVulnerability) bool {
	return vuln.GetCvssV3().GetAttackVector() == storage.CVSSV3_ATTACK_NETWORK || vuln.GetCvssV2().GetAttackVector() == storage.CVSSV2_ATTACK_NETWORK
}

func minFloat32(a, b float32) float32 {
	return float32(math.Min(float64(a), float64(b)))
}

// countDistinctCVEs includes unknown-severity findings, which do not affect the risk score.
func countDistinctCVEs(components []*storage.EmbeddedImageScanComponent) int {
	cves := set.NewStringSet()
	for _, component := range components {
		for _, vuln := range component.GetVulns() {
			if cve := vuln.GetCve(); cve != "" {
				cves.Add(cve)
			}
		}
	}
	return cves.Cardinality()
}
