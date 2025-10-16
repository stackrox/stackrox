package internaltov2storage

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

// ComplianceOperatorSuite converts message from sensor to storage message
func ComplianceOperatorSuite(sensorData *central.ComplianceOperatorSuiteV2, clusterID string) *storage.ComplianceOperatorSuiteV2 {
	sensorStatus := sensorData.GetStatus()
	status := &storage.ComplianceOperatorStatus{}
	status.SetPhase(sensorStatus.GetPhase())
	status.SetResult(sensorStatus.GetResult())
	status.SetErrorMessage(sensorStatus.GetErrorMessage())
	status.SetConditions(getConditions(sensorStatus.GetConditions()))

	cosv2 := &storage.ComplianceOperatorSuiteV2{}
	cosv2.SetId(sensorData.GetId())
	cosv2.SetName(sensorData.GetName())
	cosv2.SetClusterId(clusterID)
	cosv2.SetStatus(status)
	return cosv2
}
