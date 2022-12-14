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

// NodeInventoryCollector is the implementation of NodeInventorizer
type NodeInventoryCollector struct {
}

// Scan scans the current node and returns the results as storage.NodeInventory object
func (n *NodeInventoryCollector) Scan(nodeName string) (*storage.NodeInventory, error) {
	componentsHost, err := nodes.Analyze(nodeName, "/host/", false)
	log.Info("Finished node inventory /host scan")
	if err != nil {
		log.Errorf("Error scanning node /host inventory: %v", err)
		return nil, err
	}
	log.Infof("Components found under /host: %v", componentsHost)

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
	v1rhelc := make([]*scannerV1.RHELComponent, len(rc.Packages))
	for _, rhelc := range rc.Packages {
		v1rhelc = append(v1rhelc, &scannerV1.RHELComponent{
			// TODO(ROX-13936): Find out if ID field is needed here
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
