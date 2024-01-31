package internaltov2storage

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

// ComplianceOperatorSuite converts message from sensor to storage message
func ComplianceOperatorSuite(sensorData *central.ComplianceOperatorSuite, clusterID string) *storage.ComplianceOperatorSuite {
	sensorStatus := sensorData.Status
	status := &storage.ComplianceOperatorSuite_Status{
		Phase:        sensorStatus.GetPhase(),
		Result:       sensorStatus.GetResult(),
		ErrorMessage: sensorStatus.GetErrorMessage(),
		Conditions:   getConditions(sensorStatus.GetConditions()),
	}

	return &storage.ComplianceOperatorSuite{
		Id:        sensorData.GetId(),
		Name:      sensorData.GetName(),
		ClusterId: clusterID,
		Status:    status,
	}
}
