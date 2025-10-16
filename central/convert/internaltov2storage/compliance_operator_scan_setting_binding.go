package internaltov2storage

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

func getConditions(conditionsData []*central.ComplianceOperatorCondition) []*storage.ComplianceOperatorCondition {
	conditions := make([]*storage.ComplianceOperatorCondition, 0, len(conditionsData))
	for _, c := range conditionsData {
		coc := &storage.ComplianceOperatorCondition{}
		coc.SetType(c.GetType())
		coc.SetStatus(c.GetStatus())
		coc.SetMessage(c.GetMessage())
		coc.SetReason(c.GetReason())
		coc.SetLastTransitionTime(c.GetLastTransitionTime())
		conditions = append(conditions, coc)
	}
	return conditions
}

// ComplianceOperatorScanSettingBindingObject converts internal api V2 compliance scan setting binding object to a V2 storage
// compliance scan setting binding object
func ComplianceOperatorScanSettingBindingObject(sensorData *central.ComplianceOperatorScanSettingBindingV2, clusterID string) *storage.ComplianceOperatorScanSettingBindingV2 {
	cos := &storage.ComplianceOperatorStatus{}
	cos.SetPhase(sensorData.GetStatus().GetPhase())
	cos.SetConditions(getConditions(sensorData.GetStatus().GetConditions()))
	cossbv2 := &storage.ComplianceOperatorScanSettingBindingV2{}
	cossbv2.SetId(sensorData.GetId())
	cossbv2.SetName(sensorData.GetName())
	cossbv2.SetClusterId(clusterID)
	cossbv2.SetScanSettingName(sensorData.GetScanSettingName())
	cossbv2.SetProfileNames(sensorData.GetProfileNames())
	cossbv2.SetStatus(cos)
	cossbv2.SetLabels(sensorData.GetLabels())
	cossbv2.SetAnnotations(sensorData.GetAnnotations())
	return cossbv2
}
