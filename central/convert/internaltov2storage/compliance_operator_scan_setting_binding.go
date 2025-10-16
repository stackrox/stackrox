package internaltov2storage

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

func getConditions(conditionsData []*central.ComplianceOperatorCondition) []*storage.ComplianceOperatorCondition {
	conditions := make([]*storage.ComplianceOperatorCondition, 0, len(conditionsData))
	for _, c := range conditionsData {
		conditions = append(conditions, &storage.ComplianceOperatorCondition{
			Type:               c.GetType(),
			Status:             c.GetStatus(),
			Message:            c.GetMessage(),
			Reason:             c.GetReason(),
			LastTransitionTime: c.GetLastTransitionTime(),
		})
	}
	return conditions
}

// ComplianceOperatorScanSettingBindingObject converts internal api V2 compliance scan setting binding object to a V2 storage
// compliance scan setting binding object
func ComplianceOperatorScanSettingBindingObject(sensorData *central.ComplianceOperatorScanSettingBindingV2, clusterID string) *storage.ComplianceOperatorScanSettingBindingV2 {
	return &storage.ComplianceOperatorScanSettingBindingV2{
		Id:              sensorData.GetId(),
		Name:            sensorData.GetName(),
		ClusterId:       clusterID,
		ScanSettingName: sensorData.GetScanSettingName(),
		ProfileNames:    sensorData.GetProfileNames(),
		Status: &storage.ComplianceOperatorStatus{
			Phase:      sensorData.GetStatus().GetPhase(),
			Conditions: getConditions(sensorData.GetStatus().GetConditions()),
		},
		Labels:      sensorData.GetLabels(),
		Annotations: sensorData.GetAnnotations(),
	}
}
