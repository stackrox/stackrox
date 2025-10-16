package inventory

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
)

// ToNodeInventory converts a NodeInventory response to a native storage.NodeInventory version
func ToNodeInventory(resp *scannerV1.GetNodeInventoryResponse) *storage.NodeInventory {
	ni := &storage.NodeInventory{}
	// NodeId is only available to Sensor. Until it arrives there, we add a placeholder
	ni.SetNodeId(uuid.Nil.String())
	ni.SetNodeName(resp.GetNodeName())
	ni.SetScanTime(protocompat.TimestampNow())
	ni.SetComponents(toStorageComponents(resp.GetComponents()))
	ni.SetNotes(convertNotes(resp.GetNotes()))
	return ni
}

func toStorageComponents(c *scannerV1.Components) *storage.NodeInventory_Components {
	if c == nil {
		return nil
	}

	nc := &storage.NodeInventory_Components{}
	nc.SetNamespace(c.GetNamespace())
	nc.SetRhelComponents(convertRHELComponents(c.GetRhelComponents()))
	nc.SetRhelContentSets(c.GetRhelContentSets())
	return nc
}

func convertRHELComponents(rhelComponents []*scannerV1.RHELComponent) []*storage.NodeInventory_Components_RHELComponent {
	if rhelComponents == nil {
		return nil
	}
	convertedComponents := make([]*storage.NodeInventory_Components_RHELComponent, 0, len(rhelComponents))
	for _, c := range rhelComponents {
		sc := &storage.NodeInventory_Components_RHELComponent{}
		sc.SetId(c.GetId())
		sc.SetName(c.GetName())
		sc.SetNamespace(c.GetNamespace())
		sc.SetVersion(c.GetVersion())
		sc.SetArch(c.GetArch())
		sc.SetModule(c.GetModule())
		sc.SetAddedBy(c.GetAddedBy())
		sc.SetExecutables(convertExecutables(c.GetExecutables()))
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
		ncre := &storage.NodeInventory_Components_RHELComponent_Executable{}
		ncre.SetPath(e.GetPath())
		ncre.SetRequiredFeatures(convertRequiredFeatures(e.GetRequiredFeatures()))
		convertedExecutables = append(convertedExecutables, ncre)
	}
	return convertedExecutables
}

func convertRequiredFeatures(features []*scannerV1.FeatureNameVersion) []*storage.NodeInventory_Components_RHELComponent_Executable_FeatureNameVersion {
	if features == nil {
		return nil
	}
	convertedFeatures := make([]*storage.NodeInventory_Components_RHELComponent_Executable_FeatureNameVersion, 0, len(features))
	for _, f := range features {
		ncref := &storage.NodeInventory_Components_RHELComponent_Executable_FeatureNameVersion{}
		ncref.SetName(f.GetName())
		ncref.SetVersion(f.GetVersion())
		convertedFeatures = append(convertedFeatures, ncref)
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
