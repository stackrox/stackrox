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

type classificationResult struct {
	priority storage.RiskPriorityClassification
	activity storage.RiskActivityClassification
	exposure storage.RiskExposureClassification
	posture  storage.RiskPostureClassification
}

func classifyRisk(inputs classificationInputs) classificationResult {
	activity := classifyActivity(inputs)
	exposure := classifyExposure(inputs)
	posture := classifyPosture(inputs)

	return classificationResult{
		priority: derivePriorityClassification(activity, exposure, posture),
		activity: activity,
		exposure: exposure,
		posture:  posture,
	}
}

func derivePriorityClassification(activity storage.RiskActivityClassification, exposure storage.RiskExposureClassification, posture storage.RiskPostureClassification) storage.RiskPriorityClassification {
	switch activity {
	case storage.RiskActivityClassification_RISK_ACTIVITY_RESPOND_NOW:
		return storage.RiskPriorityClassification_RISK_PRIORITY_RESPOND_NOW
	case storage.RiskActivityClassification_RISK_ACTIVITY_INVESTIGATE_NOW:
		return storage.RiskPriorityClassification_RISK_PRIORITY_INVESTIGATE_NOW
	}

	switch exposure {
	case storage.RiskExposureClassification_RISK_EXPOSURE_REMEDIATE_NOW:
		return storage.RiskPriorityClassification_RISK_PRIORITY_REMEDIATE_NOW
	case storage.RiskExposureClassification_RISK_EXPOSURE_REMEDIATE:
		return storage.RiskPriorityClassification_RISK_PRIORITY_REMEDIATE
	}

	switch posture {
	case storage.RiskPostureClassification_RISK_POSTURE_CRITICAL,
		storage.RiskPostureClassification_RISK_POSTURE_HIGH,
		storage.RiskPostureClassification_RISK_POSTURE_MEDIUM,
		storage.RiskPostureClassification_RISK_POSTURE_LOW:
		return storage.RiskPriorityClassification_RISK_PRIORITY_IMPROVE_POSTURE
	}

	return storage.RiskPriorityClassification_RISK_PRIORITY_UNKNOWN
}

func classifyActivity(inputs classificationInputs) storage.RiskActivityClassification {
	if hasConfirmedThreatPolicyViolation(inputs.riskResults) {
		return storage.RiskActivityClassification_RISK_ACTIVITY_RESPOND_NOW
	}
	if hasRiskResult(inputs.riskResults, deploymentMultiplier.ProcessBaselineHeading) || hasSuspiciousRuntimePolicyViolation(inputs.riskResults) {
		return storage.RiskActivityClassification_RISK_ACTIVITY_INVESTIGATE_NOW
	}
	return storage.RiskActivityClassification_RISK_ACTIVITY_NONE_OBSERVED
}

func classifyExposure(inputs classificationInputs) storage.RiskExposureClassification {
	external := hasExternalExposure(inputs.deployment)
	powerfulIdentity := hasPowerfulIdentity(inputs.deployment)
	privileged := hasDangerousPrivilege(inputs.deployment)
	secret := mountsSecret(inputs.deployment)
	vuln := mergedVulnerabilitySummary(inputs.imageSummaries)

	if external {
		if hasExternallyExploitableVulnerability(vuln) || (secret && hasImportantOrCriticalVulnerability(vuln)) || powerfulIdentity || privileged {
			return storage.RiskExposureClassification_RISK_EXPOSURE_REMEDIATE_NOW
		}
	}

	if powerfulIdentity || privileged || hasKnownExploitedOrCriticalVulnerability(vuln) {
		return storage.RiskExposureClassification_RISK_EXPOSURE_REMEDIATE
	}

	return storage.RiskExposureClassification_RISK_EXPOSURE_NONE_DETECTED
}

func classifyPosture(inputs classificationInputs) storage.RiskPostureClassification {
	postureScore := postureDebtScore(inputs.riskResults, inputs.imageRiskResults)
	if postureScore >= 25 {
		return storage.RiskPostureClassification_RISK_POSTURE_CRITICAL
	}
	if postureScore >= 10 {
		return storage.RiskPostureClassification_RISK_POSTURE_HIGH
	}
	if postureScore >= 3 {
		return storage.RiskPostureClassification_RISK_POSTURE_MEDIUM
	}
	if postureScore > 1 || hasAnyVulnerability(mergedVulnerabilitySummary(inputs.imageSummaries)) {
		return storage.RiskPostureClassification_RISK_POSTURE_LOW
	}
	return storage.RiskPostureClassification_RISK_POSTURE_NONE_DETECTED
}

func postureDebtScore(riskResults []*storage.Risk_Result, imageRiskResults map[string][]*storage.Risk_Result) float32 {
	var score float32 = 1
	for _, result := range riskResults {
		switch result.GetName() {
		case deploymentMultiplier.PolicyViolationsHeading:
			if !riskResultContainsConfirmedThreatOrSuspiciousRuntime(result) {
				score *= riskResultPostureScore(result)
			}
		case deploymentMultiplier.ServiceConfigHeading:
			score *= riskResultPostureScore(result)
		}
	}
	for _, heading := range []string{imageMultiplier.VulnerabilitiesHeading, imageMultiplier.RiskyComponentCountHeading, imageMultiplier.ComponentCountHeading, imageMultiplier.ImageAgeHeading} {
		for _, result := range imageRiskResults[heading] {
			score *= riskResultPostureScore(result)
		}
	}
	return score
}

func riskResultContainsConfirmedThreatOrSuspiciousRuntime(result *storage.Risk_Result) bool {
	if result.GetName() != deploymentMultiplier.PolicyViolationsHeading {
		return false
	}
	for _, factor := range result.GetFactors() {
		message := strings.ToLower(factor.GetMessage())
		if strings.Contains(message, "cryptocurrency") ||
			strings.Contains(message, "crypto mining") ||
			strings.Contains(message, "miner") ||
			strings.Contains(message, "malware") ||
			strings.Contains(message, "nmap") ||
			strings.Contains(message, "netcat") ||
			strings.Contains(message, "shell spawned") ||
			strings.Contains(message, "secure shell") ||
			strings.Contains(message, "iptables") ||
			strings.Contains(message, "password binaries") ||
			strings.Contains(message, "linux user add") ||
			strings.Contains(message, "linux group add") ||
			strings.Contains(message, "shadow file") ||
			strings.Contains(message, "kubernetes service endpoint") ||
			strings.Contains(message, "kubelet endpoint") {
			return true
		}
	}
	return false
}

func riskResultPostureScore(result *storage.Risk_Result) float32 {
	if result.GetScore() > 0 {
		return result.GetScore()
	}
	return riskResultLegacyScore(result)
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
	return hasPolicyViolationMatching(results, func(message string) bool {
		return strings.Contains(message, "severity: Critical")
	})
}

func hasConfirmedThreatPolicyViolation(results []*storage.Risk_Result) bool {
	return hasPolicyViolationMatching(results, func(message string) bool {
		lowerMessage := strings.ToLower(message)
		return strings.Contains(lowerMessage, "cryptocurrency") ||
			strings.Contains(lowerMessage, "crypto mining") ||
			strings.Contains(lowerMessage, "miner") ||
			strings.Contains(lowerMessage, "malware")
	})
}

func hasSuspiciousRuntimePolicyViolation(results []*storage.Risk_Result) bool {
	return hasPolicyViolationMatching(results, func(message string) bool {
		lowerMessage := strings.ToLower(message)
		return strings.Contains(lowerMessage, "nmap") ||
			strings.Contains(lowerMessage, "netcat") ||
			strings.Contains(lowerMessage, "shell spawned") ||
			strings.Contains(lowerMessage, "secure shell") ||
			strings.Contains(lowerMessage, "iptables") ||
			strings.Contains(lowerMessage, "password binaries") ||
			strings.Contains(lowerMessage, "linux user add") ||
			strings.Contains(lowerMessage, "linux group add") ||
			strings.Contains(lowerMessage, "shadow file") ||
			strings.Contains(lowerMessage, "kubernetes service endpoint") ||
			strings.Contains(lowerMessage, "kubelet endpoint")
	})
}

func hasPolicyViolationMatching(results []*storage.Risk_Result, matches func(string) bool) bool {
	for _, result := range results {
		if result.GetName() != deploymentMultiplier.PolicyViolationsHeading {
			continue
		}
		for _, factor := range result.GetFactors() {
			if matches(factor.GetMessage()) {
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

func hasExternallyExploitableVulnerability(summary *storage.RiskVulnerabilitySummary) bool {
	if summary == nil {
		return false
	}
	return hasKnownExploitedOrCriticalVulnerability(summary) || summary.GetMaxEpssPercentile() >= highEPSSPercentileThreshold
}

func hasKnownExploitedOrCriticalVulnerability(summary *storage.RiskVulnerabilitySummary) bool {
	if summary == nil {
		return false
	}
	return summary.GetCisaKev() || summary.GetKnownRansomwareCampaign() || summary.GetCriticalCount() > 0
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
