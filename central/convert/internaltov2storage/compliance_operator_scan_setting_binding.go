package internaltov2storage

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

func getConditions(conditionsData []*central.ComplianceOperatorCondition) []*storage.ComplianceOperatorCondition {
	conditions := make([]*storage.ComplianceOperatorCondition, 0, len(conditionsData))
	for _, c := range conditionsData {
		conditionType := c.GetType()
		status := c.GetStatus()
		message := c.GetMessage()
		reason := c.GetReason()
		lastTransitionTime := c.GetLastTransitionTime()

		condition := storage.ComplianceOperatorCondition_builder{
			Type:               &conditionType,
			Status:             &status,
			Message:            &message,
			Reason:             &reason,
			LastTransitionTime: lastTransitionTime,
		}.Build()
		conditions = append(conditions, condition)
	}
	return conditions
}

// ComplianceOperatorScanSettingBindingObject converts internal api V2 compliance scan setting binding object to a V2 storage
// compliance scan setting binding object
func ComplianceOperatorScanSettingBindingObject(sensorData *central.ComplianceOperatorScanSettingBindingV2, clusterID string) *storage.ComplianceOperatorScanSettingBindingV2 {
	id := sensorData.GetId()
	name := sensorData.GetName()
	scanSettingName := sensorData.GetScanSettingName()
	profileNames := sensorData.GetProfileNames()
	labels := sensorData.GetLabels()
	annotations := sensorData.GetAnnotations()

	phase := sensorData.GetStatus().GetPhase()
	conditions := getConditions(sensorData.GetStatus().GetConditions())
	status := storage.ComplianceOperatorStatus_builder{
		Phase:      &phase,
		Conditions: conditions,
	}.Build()

	return storage.ComplianceOperatorScanSettingBindingV2_builder{
		Id:              &id,
		Name:            &name,
		ClusterId:       &clusterID,
		ScanSettingName: &scanSettingName,
		ProfileNames:    profileNames,
		Status:          status,
		Labels:          labels,
		Annotations:     annotations,
	}.Build()
}
