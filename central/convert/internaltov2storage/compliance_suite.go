package internaltov2storage

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

// ComplianceOperatorSuite converts message from sensor to storage message
func ComplianceOperatorSuite(sensorData *central.ComplianceOperatorSuiteV2, clusterID string) *storage.ComplianceOperatorSuiteV2 {
	sensorStatus := sensorData.GetStatus()
	phase := sensorStatus.GetPhase()
	result := sensorStatus.GetResult()
	errorMessage := sensorStatus.GetErrorMessage()
	conditions := getConditions(sensorStatus.GetConditions())

	status := storage.ComplianceOperatorStatus_builder{
		Phase:        &phase,
		Result:       &result,
		ErrorMessage: &errorMessage,
		Conditions:   conditions,
	}.Build()

	id := sensorData.GetId()
	name := sensorData.GetName()

	return storage.ComplianceOperatorSuiteV2_builder{
		Id:        &id,
		Name:      &name,
		ClusterId: &clusterID,
		Status:    status,
	}.Build()
}
