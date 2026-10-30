package deployment

import (
	"strings"

	deploymentMultiplier "github.com/stackrox/rox/central/risk/multipliers/deployment"
	imageMultiplier "github.com/stackrox/rox/central/risk/multipliers/image"
	"github.com/stackrox/rox/generated/storage"
)

const highEPSSPercentileThreshold = 0.95

type classificationInputs struct {
	deployment       *storage.Deployment
	riskResults      []*storage.Risk_Result
	imageSummaries   []*storage.RiskVulnerabilitySummary
	imageRiskResults map[string][]*storage.Risk_Result
}

func classifyRisk(inputs classificationInputs) storage.RiskPriorityClassification {
	if hasCriticalPolicyViolation(inputs.riskResults) {
		return storage.RiskPriorityClassification_RISK_PRIORITY_RESPOND_NOW
	}
	if hasRiskResult(inputs.riskResults, deploymentMultiplier.ProcessBaselineHeading) {
		return storage.RiskPriorityClassification_RISK_PRIORITY_INVESTIGATE_NOW
	}

	external := hasExternalExposure(inputs.deployment)
	powerfulIdentity := hasPowerfulIdentity(inputs.deployment)
	privileged := hasDangerousPrivilege(inputs.deployment)
	secret := mountsSecret(inputs.deployment)
	vuln := mergedVulnerabilitySummary(inputs.imageSummaries)
	if external {
		if hasKnownExploitedOrCriticalVulnerability(vuln) || (secret && hasImportantOrCriticalVulnerability(vuln)) || powerfulIdentity || privileged {
			return storage.RiskPriorityClassification_RISK_PRIORITY_REMEDIATE_NOW
		}
	}

	if powerfulIdentity || privileged || hasImportantOrCriticalVulnerability(vuln) {
		return storage.RiskPriorityClassification_RISK_PRIORITY_REMEDIATE
	}
	if len(inputs.riskResults) != 0 || hasAnyVulnerability(vuln) || hasImagePostureRisk(inputs.imageRiskResults) {
		return storage.RiskPriorityClassification_RISK_PRIORITY_IMPROVE_POSTURE
	}
	return storage.RiskPriorityClassification_RISK_PRIORITY_UNKNOWN
}

func hasRiskResult(results []*storage.Risk_Result, name string) bool {
	for _, result := range results {
		if result.GetName() == name {
			return true
		}
	}
	return false
}

func hasCriticalPolicyViolation(results []*storage.Risk_Result) bool {
	for _, result := range results {
		if result.GetName() != deploymentMultiplier.PolicyViolationsHeading {
			continue
		}
		for _, factor := range result.GetFactors() {
			if strings.Contains(factor.GetMessage(), "severity: Critical") {
				return true
			}
		}
	}
	return false
}

func hasExternalExposure(deployment *storage.Deployment) bool {
	for _, port := range deployment.GetPorts() {
		switch port.GetExposure() {
		case storage.PortConfig_EXTERNAL, storage.PortConfig_ROUTE:
			return true
		}
	}
	return false
}

func hasPowerfulIdentity(deployment *storage.Deployment) bool {
	switch deployment.GetServiceAccountPermissionLevel() {
	case storage.PermissionLevel_ELEVATED_CLUSTER_WIDE, storage.PermissionLevel_CLUSTER_ADMIN:
		return deployment.GetAutomountServiceAccountToken()
	default:
		return false
	}
}

func hasDangerousPrivilege(deployment *storage.Deployment) bool {
	if deployment.GetHostNetwork() || deployment.GetHostPid() || deployment.GetHostIpc() {
		return true
	}
	for _, container := range deployment.GetContainers() {
		securityContext := container.GetSecurityContext()
		if securityContext.GetPrivileged() {
			return true
		}
		for _, capability := range securityContext.GetAddCapabilities() {
			switch strings.ToUpper(capability) {
			case "SYS_ADMIN", "NET_ADMIN", "SYS_MODULE":
				return true
			}
		}
	}
	return false
}

func mountsSecret(deployment *storage.Deployment) bool {
	for _, container := range deployment.GetContainers() {
		if len(container.GetSecrets()) != 0 {
			return true
		}
	}
	return false
}

func mergedVulnerabilitySummary(summaries []*storage.RiskVulnerabilitySummary) *storage.RiskVulnerabilitySummary {
	merged := &storage.RiskVulnerabilitySummary{}
	for _, summary := range summaries {
		if summary == nil {
			continue
		}
		merged.CriticalCount += summary.GetCriticalCount()
		merged.ImportantCount += summary.GetImportantCount()
		merged.ModerateCount += summary.GetModerateCount()
		merged.LowCount += summary.GetLowCount()
		merged.CisaKev = merged.GetCisaKev() || summary.GetCisaKev()
		merged.KnownRansomwareCampaign = merged.GetKnownRansomwareCampaign() || summary.GetKnownRansomwareCampaign()
		merged.NetworkAttackVector = merged.GetNetworkAttackVector() || summary.GetNetworkAttackVector()
		if summary.GetMaxEpssPercentile() > merged.GetMaxEpssPercentile() {
			merged.MaxEpssPercentile = summary.GetMaxEpssPercentile()
		}
		if vulnerabilitySeverityRank(summary.GetMaxSeverity()) > vulnerabilitySeverityRank(merged.GetMaxSeverity()) {
			merged.MaxSeverity = summary.GetMaxSeverity()
		}
	}
	if !hasAnyVulnerability(merged) {
		return nil
	}
	return merged
}

func hasKnownExploitedOrCriticalVulnerability(summary *storage.RiskVulnerabilitySummary) bool {
	if summary == nil {
		return false
	}
	return summary.GetCisaKev() || summary.GetKnownRansomwareCampaign() || summary.GetMaxEpssPercentile() >= highEPSSPercentileThreshold || summary.GetCriticalCount() > 0
}

func hasImportantOrCriticalVulnerability(summary *storage.RiskVulnerabilitySummary) bool {
	if summary == nil {
		return false
	}
	return summary.GetCriticalCount() > 0 || summary.GetImportantCount() > 0
}

func hasAnyVulnerability(summary *storage.RiskVulnerabilitySummary) bool {
	if summary == nil {
		return false
	}
	return summary.GetCriticalCount()+summary.GetImportantCount()+summary.GetModerateCount()+summary.GetLowCount() > 0
}

func hasImagePostureRisk(imageRiskResults map[string][]*storage.Risk_Result) bool {
	for _, heading := range []string{imageMultiplier.RiskyComponentCountHeading, imageMultiplier.ComponentCountHeading, imageMultiplier.ImageAgeHeading, imageMultiplier.VulnerabilitiesHeading} {
		if len(imageRiskResults[heading]) != 0 {
			return true
		}
	}
	return false
}

func vulnerabilitySeverityRank(severity storage.VulnerabilitySeverity) int {
	switch severity {
	case storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY:
		return 1
	case storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY:
		return 2
	case storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY:
		return 3
	case storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY:
		return 4
	default:
		return 0
	}
}
