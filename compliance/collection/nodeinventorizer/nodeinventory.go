package nodeinventorizer

import (
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/scanner/database"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stackrox/scanner/pkg/analyzer/nodes"
)

// NodeInventorizer is the interface that defines the interface a scanner must implement
type NodeInventorizer interface {
	Scan(nodeName string) (*storage.NodeInventory, error)
}

// NodeInventoryCollector is an implementation of NodeInventorizer
type NodeInventoryCollector struct {
}

// Scan scans the current node and returns the results as storage.NodeInventory object
func (n *NodeInventoryCollector) Scan(nodeName string) (*storage.NodeInventory, error) {
	log.Info("Started node inventory")
	// uncertifiedRHEL is set to false, as scans are only supported on RHCOS for now,
	// which only exists in certified versions
	componentsHost, err := nodes.Analyze(nodeName, "/host/", false)
	log.Info("Finished node inventory")
	if err != nil {
		log.Errorf("Error scanning node /host inventory: %v", err)
		return nil, err
	}
	log.Debugf("Components found under /host: %v", componentsHost)

	protoComponents := protoComponentsFromScanComponents(componentsHost)

	m := &storage.NodeInventory{
		NodeName:   nodeName,
		ScanTime:   timestamp.TimestampNow(),
		Components: protoComponents,
	}
	return m, nil
}

func protoComponentsFromScanComponents(c *nodes.Components) *scannerV1.Components {
	if c == nil {
		return nil
	}

	// For now, we only care about RHEL components, but this must be extended once we support non-RHCOS
	rhelComponents := convertRHELComponents(c.CertifiedRHELComponents)

	protoComponents := &scannerV1.Components{
		Namespace:          c.OSNamespace.Name,
		OsComponents:       nil,
		RhelComponents:     rhelComponents,
		LanguageComponents: nil,
	}
	return protoComponents
}

func convertRHELComponents(rc *database.RHELv2Components) []*scannerV1.RHELComponent {
	if rc == nil || rc.Packages == nil {
		log.Warn("No RHEL packages found in scan result")
		return nil
	}

	seenEntries := make(map[int]*database.RHELv2Package)
	v1rhelc := make([]*scannerV1.RHELComponent, 0, len(rc.Packages))

	// referencing this label by the inner `continue` lets us skip the rest of the outer loop as well,
	// resulting in the colliding component not being added to v1rhelc.
COMPONENTS:
	for i, rhelc := range rc.Packages {
		for _, candidate := range seenEntries {
			if equalRHELv2Packages(candidate, rhelc) {
				log.Warnf("Detected package collision in Node Inventory scan. Skipping package %v for id %v", candidate, i)
				continue COMPONENTS
			}
		}
		// If the entry didn't produce a collision, add it to seenEntries and result
		seenEntries[i] = rhelc
		log.Debugf("Adding component %v to v1rhelc", rhelc.Name)
		v1rhelc = append(v1rhelc, &scannerV1.RHELComponent{
			Id:          int64(i),
			Name:        rhelc.Name,
			Namespace:   rc.Dist,
			Version:     rhelc.Version,
			Arch:        rhelc.Arch,
			Module:      rhelc.Module,
			Cpes:        rc.CPEs,
			Executables: rhelc.Executables,
		})
	}
	return v1rhelc
}

func equalRHELv2Packages(a *database.RHELv2Package, b *database.RHELv2Package) bool {
	if a == nil || b == nil {
		return false
	}
	if a.Name == b.Name && a.Version == b.Version && a.Arch == b.Arch && a.Module == b.Module {
		return true
	}
	return false
}
