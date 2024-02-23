package internaltov2storage

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

// ComplianceOperatorSuite converts message from sensor to storage message
func ComplianceOperatorSuite(sensorData *central.ComplianceOperatorSuiteV2, clusterID string) *storage.ComplianceOperatorSuiteV2 {
	sensorStatus := sensorData.Status
	status := &storage.ComplianceOperatorStatus{
		Phase:        sensorStatus.GetPhase(),
		Result:       sensorStatus.GetResult(),
		ErrorMessage: sensorStatus.GetErrorMessage(),
		Conditions:   getConditions(sensorStatus.GetConditions()),
	}

	return &storage.ComplianceOperatorSuiteV2{
		Id:        sensorData.GetId(),
		Name:      sensorData.GetName(),
		ClusterId: clusterID,
		Status:    status,
	}
}
