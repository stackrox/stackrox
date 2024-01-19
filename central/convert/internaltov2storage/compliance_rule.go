package internaltov2storage

import (
	"strings"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

const (
	standardsKey = "policies.open-cluster-management.io/standards"

	controlAnnotationBase = "control.compliance.openshift.io/"
)

// ComplianceOperatorRule converts message from sensor to V2 storage
func ComplianceOperatorRule(sensorData *central.ComplianceOperatorRuleV2, clusterID string) *storage.ComplianceOperatorRuleV2 {
	fixes := make([]*storage.ComplianceOperatorRuleV2_Fix, 0, len(sensorData.Fixes))
	for _, fix := range sensorData.Fixes {
		fixes = append(fixes, &storage.ComplianceOperatorRuleV2_Fix{
			Platform:   fix.GetPlatform(),
			Disruption: fix.GetDisruption(),
		})
	}

	// Get the standards with policies.open-cluster-management.io/standards so that
	// we can get the controls via control.compliance.openshift.io/STANDARD
	standards := strings.Split(sensorData.GetAnnotations()[standardsKey], ",")
	controls := make([]*storage.RuleControls, 0, len(standards))
	for _, standard := range standards {
		controls = append(controls, &storage.RuleControls{
			Standard: standard,
			Controls: strings.Split(sensorData.GetAnnotations()[controlAnnotationBase+standard], ";"),
		})
	}

	return &storage.ComplianceOperatorRuleV2{
		Id:          sensorData.GetId(),
		RuleId:      sensorData.GetRuleId(),
		Name:        sensorData.GetName(),
		RuleType:    sensorData.GetRuleType(),
		Severity:    severityToV2[sensorData.GetSeverity()],
		Labels:      sensorData.GetLabels(),
		Annotations: sensorData.GetAnnotations(),
		Title:       sensorData.GetTitle(),
		Description: sensorData.GetDescription(),
		Rationale:   sensorData.GetRationale(),
		Fixes:       fixes,
		Warning:     sensorData.GetWarning(),
		Controls:    controls,
		ClusterId:   clusterID,
	}
}
