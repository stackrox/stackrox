package internaltov1storage

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

// ComplianceOperatorRule converts message from sensor to V1 storage
func ComplianceOperatorRule(sensorData *central.ComplianceOperatorRuleV2, clusterID string) *storage.ComplianceOperatorRule {
	return &storage.ComplianceOperatorRule{
		Id:          sensorData.GetId(),
		RuleId:      sensorData.GetRuleId(),
		Name:        sensorData.GetName(),
		ClusterId:   clusterID,
		Labels:      sensorData.GetLabels(),
		Annotations: sensorData.GetAnnotations(),
		Title:       sensorData.GetTitle(),
		Description: sensorData.GetDescription(),
		Rationale:   sensorData.GetRationale(),
	}
}
