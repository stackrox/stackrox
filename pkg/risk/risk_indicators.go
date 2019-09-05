package risk

import "github.com/stackrox/rox/generated/storage"

// All risk indicators used to compute risk of risk entities
var (
	PolicyViolations     = newRiskIndicator("Policy Violations", 1, storage.RiskEntityType_DEPLOYMENT)
	SuspiciousProcesses  = newRiskIndicator("Suspicious Process Executions", 2, storage.RiskEntityType_DEPLOYMENT)
	ImageVulnerabilities = newRiskIndicator("Image Vulnerabilities", 3, storage.RiskEntityType_IMAGE)
	ServiceConfiguration = newRiskIndicator("Service Configuration", 4, storage.RiskEntityType_DEPLOYMENT)
	PortExposure         = newRiskIndicator("Service Reachability", 5, storage.RiskEntityType_DEPLOYMENT)
	RiskyImageComponent  = newRiskIndicator("Components Useful for Attackers", 6, storage.RiskEntityType_IMAGE)
	ImageComponentCount  = newRiskIndicator("Number of Components in Image", 7, storage.RiskEntityType_IMAGE)
	ImageAge             = newRiskIndicator("Image Freshness", 8, storage.RiskEntityType_IMAGE)
	RBACConfiguration    = newRiskIndicator("RBAC Configuration", 9, storage.RiskEntityType_SERVICEACCOUNT)

	AllIndicatorMap = make(map[string]Indicator)
)

// Indicator contains metadata about the risk indicator
type Indicator struct {
	DisplayTitle    string
	DisplayPriority int32
	Description     string
	EntityAppliedTo storage.RiskEntityType
}

func newRiskIndicator(displayTitle string, displayPriority int32, entityAppliedTo storage.RiskEntityType) Indicator {
	indicator := Indicator{
		DisplayTitle:    displayTitle,
		DisplayPriority: displayPriority,
		EntityAppliedTo: entityAppliedTo,
	}

	AllIndicatorMap[displayTitle] = indicator
	return indicator
}
