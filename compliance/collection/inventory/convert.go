package inventory

import (
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
)

// ToNodeInventory converts a NodeInventory response to a native storage.NodeInventory version
func ToNodeInventory(resp *scannerV1.GetNodeInventoryResponse) *storage.NodeInventory {
	return &storage.NodeInventory{
		// NodeId is only available to Sensor. Until it arrives there, we add a placeholder
		NodeId:     uuid.Nil.String(),
		NodeName:   resp.GetNodeName(),
		ScanTime:   timestamp.TimestampNow(),
		Components: toStorageComponents(resp.GetComponents()),
		Notes:      convertNotes(resp.GetNotes()),
	}
}

func toStorageComponents(c *scannerV1.Components) *storage.NodeInventory_Components {
	if c == nil {
		return nil
	}

	return &storage.NodeInventory_Components{
		Namespace:       c.GetNamespace(),
		RhelComponents:  convertRHELComponents(c.GetRhelComponents()),
		RhelContentSets: c.GetRhelContentSets(),
	}
}

func convertRHELComponents(rhelComponents []*scannerV1.RHELComponent) []*storage.NodeInventory_Components_RHELComponent {
	if rhelComponents == nil {
		return nil
	}
	convertedComponents := make([]*storage.NodeInventory_Components_RHELComponent, 0, len(rhelComponents))
	for _, c := range rhelComponents {
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

func convertExecutables(executables []*scannerV1.Executable) []*storage.NodeInventory_Components_RHELComponent_Executable {
	if executables == nil {
		return nil
	}
	convertedExecutables := make([]*storage.NodeInventory_Components_RHELComponent_Executable, 0, len(executables))
	for _, e := range executables {
		convertedExecutables = append(convertedExecutables, &storage.NodeInventory_Components_RHELComponent_Executable{
			Path:             e.GetPath(),
			RequiredFeatures: convertRequiredFeatures(e.GetRequiredFeatures()),
		})
	}
	return convertedExecutables
}

func convertRequiredFeatures(features []*scannerV1.FeatureNameVersion) []*storage.NodeInventory_Components_RHELComponent_Executable_FeatureNameVersion {
	if features == nil {
		return nil
	}
	convertedFeatures := make([]*storage.NodeInventory_Components_RHELComponent_Executable_FeatureNameVersion, 0, len(features))
	for _, f := range features {
		convertedFeatures = append(convertedFeatures, &storage.NodeInventory_Components_RHELComponent_Executable_FeatureNameVersion{
			Name:    f.GetName(),
			Version: f.GetVersion(),
		})
	}
	return convertedFeatures
}

func convertNotes(notes []scannerV1.Note) []storage.NodeInventory_Note {
	if notes == nil {
		return nil
	}
	convertedNotes := make([]storage.NodeInventory_Note, 0, len(notes))
	for _, n := range notes {
		switch n {
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
