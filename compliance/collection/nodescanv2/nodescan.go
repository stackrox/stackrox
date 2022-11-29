package nodescanv2

import (
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/scanner/database"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stackrox/scanner/pkg/analyzer/nodes"
)

// NodeScanner defines an interface for V2 NodeScanning
type NodeScanner interface {
	Scan(nodeName string) (*storage.NodeInventory, error)
}

// NodeScan is the V2 NodeScanning implementation
type NodeScan struct {
}

// Scan scans the current node and returns the results as storage.NodeInventory object
func (n *NodeScan) Scan(nodeName string) (*storage.NodeInventory, error) {
	componentsHost, err := nodes.Analyze(nodeName, "/host/", false)
	log.Info("Finished node inventory /host scan")
	if err != nil {
		log.Errorf("Error scanning node /host inventory: %v", err)
	}
	log.Infof("Components found under /host: %v", componentsHost)
	if err != nil {
		return nil, err
	}

	var protoComponents *scannerV1.Components
	if componentsHost != nil {
		protoComponents = protoComponentsFromScanComponents(componentsHost)
	}
	m := &storage.NodeInventory{
		NodeName:   nodeName,
		ScanTime:   timestamp.TimestampNow(),
		Components: protoComponents,
	}
	return m, nil
}

func protoComponentsFromScanComponents(c *nodes.Components) *scannerV1.Components {
	var components []*scannerV1.RHELComponent
	// For now, we only care about RHEL components, but this must be extended once we support non-RHCOS
	if c.CertifiedRHELComponents != nil {
		components = convertRHELComponents(c.CertifiedRHELComponents)
	}
	pc := scannerV1.Components{
		Namespace:          c.OSNamespace.Name,
		OsComponents:       nil,
		RhelComponents:     components,
		LanguageComponents: nil,
	}
	return &pc
}

func convertRHELComponents(rc *database.RHELv2Components) []*scannerV1.RHELComponent {
	v1rhelc := make([]*scannerV1.RHELComponent, 0)
	if rc.Packages == nil {
		log.Warn("No RHEL packages found in scan result")
		return v1rhelc
	}
	for _, rhelc := range rc.Packages {
		v1rhelc = append(v1rhelc, &scannerV1.RHELComponent{
			Id:          0,
			Name:        rhelc.Name,
			Namespace:   rc.Dist, // check
			Version:     rhelc.Version,
			Arch:        rhelc.Arch,
			Module:      rhelc.Module,
			Cpes:        rc.CPEs, // do we just append all here?
			Executables: rhelc.Executables,
			// AddedBy:     "",                // do we know?
		})
	}
	return v1rhelc
}
