package internaltostorage

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

func convertSource(val central.SourceType) storage.SourceType {
	return storage.SourceType(central.SourceType_value[val.String()])
}

func convertComponents(components []*central.VirtualMachineComponent) []*storage.EmbeddedImageScanComponent {
	converted := []*storage.EmbeddedImageScanComponent{}
	for _, c := range components {
		converted = append(converted, &storage.EmbeddedImageScanComponent{
			Architecture: c.GetArchitecture(),
			License: &storage.License{
				Name: c.GetLicense().GetName(),
				Type: c.GetLicense().GetType(),
				Url:  c.GetLicense().GetUrl(),
			},
			Location: c.GetLocation(),
			Name:     c.GetName(),
			Source:   convertSource(c.GetSource()),
			Version:  c.GetVersion(),
		})
	}
	return converted
}

// VirtualMachine converts an internalapi Central message to a storage message.
func VirtualMachine(virtualMachine *central.VirtualMachine) *storage.VirtualMachine {
	return &storage.VirtualMachine{
		ClusterId:   virtualMachine.GetClusterId(),
		ClusterName: virtualMachine.GetClusterName(),
		Facts:       virtualMachine.GetFacts(),
		Id:          virtualMachine.GetId(),
		Name:        virtualMachine.GetName(),
		Namespace:   virtualMachine.GetNamespace(),
		Scan: &storage.VirtualMachineScan{
			Components: convertComponents(virtualMachine.GetComponents()),
		},
	}
}
