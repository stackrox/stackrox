package sensortocentral

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
)

func VirtualMachine(virtualMachine *sensor.VirtualMachine) *central.VirtualMachine {
	return &central.VirtualMachine{
		ClusterId:   virtualMachine.GetClusterId(),
		ClusterName: virtualMachine.GetClusterName(),
		Facts:       virtualMachine.GetFacts(),
		Id:          virtualMachine.GetId(),
		Name:        virtualMachine.GetName(),
		Namespace:   virtualMachine.GetNamespace(),
	}
}
