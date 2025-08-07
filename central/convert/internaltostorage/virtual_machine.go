package internaltostorage

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

// ComplianceOperatorSuite converts message from sensor to storage message
func VirtualMachine(sensorData *central.VirtualMachine) *storage.VirtualMachine {
	return &storage.VirtualMachine{
		ClusterId:   sensorData.GetClusterId(),
		ClusterName: sensorData.GetClusterName(),
		Facts:       sensorData.GetFacts(),
		Id:          sensorData.GetId(),
		Name:        sensorData.GetName(),
		Namespace:   sensorData.GetNamespace(),
	}
}
