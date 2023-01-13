package nodeinventorizer

import (
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/scanner/database"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stackrox/scanner/pkg/analyzer/detection"
	"github.com/stackrox/scanner/pkg/analyzer/nodes"
	"golang.org/x/exp/maps"
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
	componentsHost, err := nodes.Analyze(nodeName, "/host/", detection.DetectComponentOpts{UncertifiedRHEL: false, IsRHCOSRequired: true})
	if err != nil {
		log.Errorf("Error scanning node /host inventory: %v", err)
		return nil, err
	}
	log.Info("Finished node inventory")
	log.Debugf("Components found under /host: %v", componentsHost)

	protoComponents := protoComponentsFromScanComponents(componentsHost)

	if protoComponents == nil {
		log.Warn("Empty components returned from NodeInventory")
	}

	// uncertifiedRHEL is false since scanning is only supported on RHCOS for now,
	// which only exists in certified versions. Therefore, no specific notes needed
	// if uncertifiedRHEL can be true in the future, we can add Note_CERTIFIED_RHEL_SCAN_UNAVAILABLE
	m := &storage.NodeInventory{
		NodeName:   nodeName,
		ScanTime:   timestamp.TimestampNow(),
		Components: protoComponents,
		Notes:      []scannerV1.Note{scannerV1.Note_LANGUAGE_CVES_UNAVAILABLE},
	}

	return m, nil
}

func protoComponentsFromScanComponents(c *nodes.Components) *scannerV1.Components {
	if c == nil {
		return nil
	}

	var namespace string
	if c.OSNamespace == nil {
		namespace = "unknown"
		// TODO(ROX-14186): Also set a note here that this is an uncertified scan
	} else {
		namespace = c.OSNamespace.Name
	}

	// For now, we only care about RHEL components, but this must be extended once we support non-RHCOS
	rhelComponents := convertAndDedupRHELComponents(c.CertifiedRHELComponents)

	protoComponents := &scannerV1.Components{
		Namespace:          namespace,
		OsComponents:       nil,
		RhelComponents:     rhelComponents,
		LanguageComponents: nil,
	}
	return protoComponents
}

func convertAndDedupRHELComponents(rc *database.RHELv2Components) []*scannerV1.RHELComponent {
	if rc == nil || rc.Packages == nil {
		log.Warn("No RHEL packages found in scan result")
		return nil
	}

	convertedComponents := make(map[string]*scannerV1.RHELComponent, 0)
	for i, rhelc := range rc.Packages {
		comp := &scannerV1.RHELComponent{
			// The loop index is used as ID, as this field only needs to be unique for each NodeInventory result slice
			Id:          int64(i),
			Name:        rhelc.Name,
			Namespace:   rc.Dist,
			Version:     rhelc.Version,
			Arch:        rhelc.Arch,
			Module:      rhelc.Module,
			Cpes:        rc.CPEs,
			Executables: rhelc.Executables,
		}
		compKey := makeComponentKey(comp)
		if compKey != "" {
			if _, contains := convertedComponents[compKey]; !contains {
				log.Debugf("Adding component %v to convertedComponents", comp.Name)
				convertedComponents[compKey] = comp
			} else {
				log.Warnf("Detected package collision in Node Inventory scan. Skipping package %s at index %d", compKey, i)
			}
		}

	}
	return maps.Values(convertedComponents)
}

func makeComponentKey(component *scannerV1.RHELComponent) string {
	return component.Name + ":" + component.Version + ":" + component.Arch + ":" + component.Module
}
