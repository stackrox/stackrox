package internaltov2storage

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

func getConditions(conditionsData []*central.ComplianceOperatorScanSettingBindingV2_Condition) []*storage.ComplianceOperatorScanSettingBindingV2_Condition {
	conditions := make([]*storage.ComplianceOperatorScanSettingBindingV2_Condition, 0, len(conditionsData))
	for _, c := range conditionsData {
		conditions = append(conditions, &storage.ComplianceOperatorScanSettingBindingV2_Condition{
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
		Conditions:      getConditions(sensorData.GetConditions()),
		Labels:          sensorData.GetLabels(),
		Annotations:     sensorData.GetAnnotations(),
	}
}
