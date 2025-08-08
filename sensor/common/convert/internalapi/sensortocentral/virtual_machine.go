package sensortocentral

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
)

func convertSource(val sensor.SourceType) central.SourceType {
	return central.SourceType(sensor.SourceType_value[val.String()])
}

func convertComponents(components []*sensor.VirtualMachineComponent) []*central.VirtualMachineComponent {
	converted := []*central.VirtualMachineComponent{}
	for _, c := range components {
		converted = append(converted, &central.VirtualMachineComponent{
			Architecture: c.GetArchitecture(),
			License: &central.License{
				Name: c.GetLicense().GetName(),
				Type: c.GetLicense().GetType(),
				Url:  c.GetLicense().GetUrl(),
			},
			Location:     c.GetLocation(),
			Name:    c.GetName(),
			Source:       convertSource(c.GetSource()),
			Version: c.GetVersion(),
		})
	}
	return converted
}

// VirtualMachine converts an internalapi Sensor message to an internal Central message.
func VirtualMachine(virtualMachine *sensor.VirtualMachine) *central.VirtualMachine {
	return &central.VirtualMachine{
		ClusterId:   virtualMachine.GetClusterId(),
		ClusterName: virtualMachine.GetClusterName(),
		Components:  convertComponents(virtualMachine.GetComponents()),
		Facts:       virtualMachine.GetFacts(),
		Id:          virtualMachine.GetId(),
		Name:        virtualMachine.GetName(),
		Namespace:   virtualMachine.GetNamespace(),
	}
}
