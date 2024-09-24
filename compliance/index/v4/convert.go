package v4

import (
	"strconv"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
)

func ToNodeInventory(r *v4.VulnerabilityReport) *storage.NodeInventory {
	return &storage.NodeInventory{
		NodeId:         uuid.Nil.String(), // FIXME
		NodeName:       "",                // FIXME
		ScanTime:       protocompat.TimestampNow(),
		Components:     toStorageComponents(r.GetContents()),
		Notes:          toStorageNotes(r.GetNotes()),
		ScannerVersion: storage.NodeInventory_SCANNER_V4,
	}
}

func toStorageComponents(c *v4.Contents) *storage.NodeInventory_Components {
	if c == nil {
		return nil
	}

	return &storage.NodeInventory_Components{
		Namespace:       "",
		RhelComponents:  toRhelComponents(c.GetPackages()),
		RhelContentSets: nil,
	}
}

func toRhelComponents(packages []*v4.Package) []*storage.NodeInventory_Components_RHELComponent {
	if packages == nil {
		return nil
	}
	convertedComponents := make([]*storage.NodeInventory_Components_RHELComponent, 0, len(packages))
	for _, p := range packages {
		cp := &storage.NodeInventory_Components_RHELComponent{
			Id:          convertID(p.GetId()),
			Name:        p.GetName(),
			Namespace:   "", // Skip?
			Version:     p.GetVersion(),
			Arch:        p.GetArch(),
			Module:      p.GetModule(),
			AddedBy:     "", // ?
			Executables: convertExecutables(p.GetSource()),
		}
		convertedComponents = append(convertedComponents, cp)
	}
	return convertedComponents
}

func convertID(id string) int64 {
	iid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		log.Warnf("Failed to convert component ID (%s) to integer: %s", id, err)
		return 0
	}
	return iid
}

func convertExecutables(p *v4.Package) []*storage.NodeInventory_Components_RHELComponent_Executable {
	if p == nil {
		return nil
	}
	cExecutables := []*storage.NodeInventory_Components_RHELComponent_Executable{
		{
			Path:             "", // We don't have a path anymore
			RequiredFeatures: convertRequiredFeatures(p),
		},
	}
	return cExecutables
}

func convertRequiredFeatures(p *v4.Package) []*storage.NodeInventory_Components_RHELComponent_Executable_FeatureNameVersion {
	return []*storage.NodeInventory_Components_RHELComponent_Executable_FeatureNameVersion{
		{
			Name:    p.GetName(),
			Version: p.GetVersion(),
		},
	}
}

func toStorageNotes(notes []v4.VulnerabilityReport_Note) []storage.NodeInventory_Note {
	if notes == nil {
		return nil
	}
	convertedNotes := make([]storage.NodeInventory_Note, 0, len(notes))
	for _, n := range notes {
		if n == v4.VulnerabilityReport_NOTE_OS_UNKNOWN || n == v4.VulnerabilityReport_NOTE_OS_UNSUPPORTED {
			convertedNotes = append(convertedNotes, storage.NodeInventory_CERTIFIED_RHEL_SCAN_UNAVAILABLE)
		}
	}
	return convertedNotes
}
