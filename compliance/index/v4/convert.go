package v4

import (
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
)

func ToNodeInventory(r *v4.VulnerabilityReport) *storage.NodeInventory {
	return &storage.NodeInventory{
		NodeId:     uuid.Nil.String(), // FIXME
		NodeName:   "",                // FIXME
		ScanTime:   protocompat.TimestampNow(),
		Components: toStorageComponents(r.GetContents()),
		Notes:      toStorageNotes(r.GetNotes()),
	}
}

func toStorageComponents(c *v4.Contents) *storage.NodeInventory_Components {
	if c == nil {
		return nil
	}

	return &storage.NodeInventory_Components{
		Namespace:       "",
		RhelComponents:  toRhelComponents(c.GetPackages()),
		RhelContentSets: toRhelContentSets(c.GetRepositories(), c.GetEnvironments()),
	}
}

func toRhelComponents(p []*v4.Package) []*storage.NodeInventory_Components_RHELComponent {
	if p == nil {
		return nil
	}
	return []*storage.NodeInventory_Components_RHELComponent{}
}

func toRhelContentSets(r []*v4.Repository, e map[string]*v4.Environment_List) []string {
	if r == nil || e == nil {
		return nil
	}
	return []string{}
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
