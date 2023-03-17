package inventory

import (
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
)

// NodeInventoryResponseToNodeInventory converts a NodeInventory response to a native storage.NodeInventory version
func NodeInventoryResponseToNodeInventory(response *scannerV1.GetNodeInventoryResponse) *storage.NodeInventory {
	ni := &storage.NodeInventory{
		NodeId:     uuid.Nil.String(),
		NodeName:   response.GetNodeName(),
		ScanTime:   timestamp.TimestampNow(),
		Components: inventoryComponentsToStorageComponents(response.GetComponents()),
		Notes:      convertNotes(response.GetNotes()),
	}
	return ni
}

func inventoryComponentsToStorageComponents(c *scannerV1.Components) *storage.NodeInventory_Components {
	if c == nil {
		return nil
	}

	var namespace string
	if c.GetNamespace() == "" {
		namespace = "unknown"
	} else {
		namespace = c.GetNamespace()
	}

	sc := &storage.NodeInventory_Components{
		Namespace:       namespace,
		RhelComponents:  convertRHELComponents(c.GetRhelComponents()),
		RhelContentSets: c.GetRhelContentSets(),
	}
	return sc

}

func convertRHELComponents(rc []*scannerV1.RHELComponent) []*storage.NodeInventory_Components_RHELComponent {
	if rc == nil {
		return nil
	}
	convertedComponents := make([]*storage.NodeInventory_Components_RHELComponent, 0)
	for _, c := range rc {
		sc := storage.NodeInventory_Components_RHELComponent{
			Id:          c.GetId(),
			Name:        c.GetName(),
			Namespace:   c.GetNamespace(),
			Version:     c.GetVersion(),
			Arch:        c.GetArch(),
			Module:      c.GetModule(),
			AddedBy:     c.GetAddedBy(),
			Executables: convertExecutables(c.GetExecutables()),
		}
		convertedComponents = append(convertedComponents, &sc)
	}
	return convertedComponents
}

func convertExecutables(e []*scannerV1.Executable) []*storage.NodeInventory_Components_RHELComponent_Executable {
	if e == nil {
		return nil
	}
	convertedExecutables := make([]*storage.NodeInventory_Components_RHELComponent_Executable, 0)
	for _, ee := range e {
		convertedExecutables = append(convertedExecutables, &storage.NodeInventory_Components_RHELComponent_Executable{
			Path:             ee.GetPath(),
			RequiredFeatures: convertRequiredFeatures(ee.GetRequiredFeatures()),
		})
	}
	return convertedExecutables
}

func convertRequiredFeatures(r []*scannerV1.FeatureNameVersion) []*storage.NodeInventory_Components_RHELComponent_Executable_FeatureNameVersion {
	if r == nil {
		return nil
	}
	convertedFeatures := make([]*storage.NodeInventory_Components_RHELComponent_Executable_FeatureNameVersion, 0)
	for _, rr := range r {
		convertedFeatures = append(convertedFeatures, &storage.NodeInventory_Components_RHELComponent_Executable_FeatureNameVersion{
			Name:    rr.GetName(),
			Version: rr.GetVersion(),
		})
	}
	return convertedFeatures
}

func convertNotes(n []scannerV1.Note) []storage.NodeInventory_Note {
	if n == nil {
		return nil
	}
	convertedNotes := make([]storage.NodeInventory_Note, 0)
	for _, nn := range n {
		switch nn {
		case scannerV1.Note_OS_CVES_UNAVAILABLE:
			convertedNotes = append(convertedNotes, storage.NodeInventory_OS_CVES_UNAVAILABLE)
		case scannerV1.Note_OS_CVES_STALE:
			convertedNotes = append(convertedNotes, storage.NodeInventory_OS_CVES_STALE)
		case scannerV1.Note_LANGUAGE_CVES_UNAVAILABLE:
			convertedNotes = append(convertedNotes, storage.NodeInventory_LANGUAGE_CVES_UNAVAILABLE)
		case scannerV1.Note_CERTIFIED_RHEL_SCAN_UNAVAILABLE:
			convertedNotes = append(convertedNotes, storage.NodeInventory_CERTIFIED_RHEL_SCAN_UNAVAILABLE)
		}
	}
	return convertedNotes
}
