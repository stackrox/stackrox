package nodeinventorizer

import (
	"time"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/compliance/collection/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/scanner/database"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stackrox/scanner/pkg/analyzer/nodes"
	"golang.org/x/exp/maps"
)

// NodeInventorizer is the interface that defines the interface a node scanner must implement
type NodeInventorizer interface {
	Scan(nodeName string) (*storage.NodeInventory, error)
}

// NodeAnalyzer is an implementation of NodeInventorizer
type NodeAnalyzer struct {
}

// Scan scans the current node and returns the results as storage.NodeInventory struct.
func (n *NodeAnalyzer) Scan(nodeName string) (*storage.NodeInventory, error) {
	metrics.ObserveScansTotal(nodeName)
	startTime := time.Now()

	log.Debug("Starting node scan")

	// uncertifiedRHEL is set to false, as scans are only supported on RHCOS for now,
	// which only exists in certified versions
	componentsHost, err := nodes.Analyze(nodeName, "/host/", nodes.AnalyzeOpts{UncertifiedRHEL: false, IsRHCOSRequired: true})

	scanDuration := time.Since(startTime)
	metrics.ObserveScanDuration(scanDuration, nodeName, err)
	log.Debugf("Scanning the node took %f seconds", scanDuration.Seconds())

	if err != nil {
		log.Errorf("Error scanning node: %v", err)
		return nil, err
	}
	log.Debugf("Components found on host filesystem: %v", componentsHost)

	protoComponents := protoComponentsFromScanComponents(componentsHost)

	if protoComponents == nil {
		log.Warn("Empty components returned from NodeInventory")
	} else {
		log.Infof("Node inventory has been built with %d packages and %d content sets",
			len(protoComponents.GetRhelComponents()), len(protoComponents.GetRhelContentSets()))
	}

	// uncertifiedRHEL is false since scanning is only supported on RHCOS for now,
	// which only exists in certified versions. Therefore, no specific notes needed
	// if uncertifiedRHEL can be true in the future, we can add Note_CERTIFIED_RHEL_SCAN_UNAVAILABLE
	m := &storage.NodeInventory{
		NodeId:     uuid.Nil.String(), // The NodeID is not available in compliance, but only in Sensor and later on
		NodeName:   nodeName,
		ScanTime:   timestamp.TimestampNow(),
		Components: protoComponents,
		Notes:      []storage.NodeInventory_Note{storage.NodeInventory_LANGUAGE_CVES_UNAVAILABLE},
	}

	metrics.ObserveNodeInventoryScan(m)
	return m, nil
}

// TODO(ROX-14029): Move conversion function into Scanner
func protoComponentsFromScanComponents(c *nodes.Components) *storage.NodeInventory_Components {
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
	var rhelComponents []*storage.NodeInventory_Components_RHELComponent
	var contentSets []string
	if c.CertifiedRHELComponents != nil {
		rhelComponents = convertAndDedupRHELComponents(c.CertifiedRHELComponents)
		contentSets = c.CertifiedRHELComponents.ContentSets
	}

	protoComponents := &storage.NodeInventory_Components{
		Namespace:       namespace,
		RhelComponents:  rhelComponents,
		RhelContentSets: contentSets,
	}
	return protoComponents
}

// TODO(ROX-14029): Move conversion function into Scanner
func convertAndDedupRHELComponents(rc *database.RHELv2Components) []*storage.NodeInventory_Components_RHELComponent {
	if rc == nil || rc.Packages == nil {
		log.Warn("No RHEL packages found in scan result")
		return nil
	}

	convertedComponents := make(map[string]*storage.NodeInventory_Components_RHELComponent, 0)
	for i, rhelc := range rc.Packages {
		if rhelc == nil {
			continue
		}
		comp := &storage.NodeInventory_Components_RHELComponent{
			// The loop index is used as ID, as this field only needs to be unique for each NodeInventory result slice
			Id:          int64(i),
			Name:        rhelc.Name,
			Namespace:   rc.Dist,
			Version:     rhelc.Version,
			Arch:        rhelc.Arch,
			Module:      rhelc.Module,
			Executables: nil,
		}
		if rhelc.Executables != nil {
			comp.Executables = convertExecutables(rhelc.Executables)
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

// TODO(ROX-14029): Move conversion function into Scanner
func convertExecutables(exe []*scannerV1.Executable) []*storage.NodeInventory_Components_RHELComponent_Executable {
	arr := make([]*storage.NodeInventory_Components_RHELComponent_Executable, len(exe))
	for i, executable := range exe {
		arr[i] = &storage.NodeInventory_Components_RHELComponent_Executable{
			Path:             executable.GetPath(),
			RequiredFeatures: nil,
		}
		if executable.GetRequiredFeatures() != nil {
			arr[i].RequiredFeatures = make([]*storage.NodeInventory_Components_RHELComponent_Executable_FeatureNameVersion, len(executable.GetRequiredFeatures()))
			for i2, fnv := range executable.GetRequiredFeatures() {
				arr[i].RequiredFeatures[i2] = &storage.NodeInventory_Components_RHELComponent_Executable_FeatureNameVersion{
					Name:    fnv.GetName(),
					Version: fnv.GetVersion(),
				}
			}
		}
	}
	return arr
}

func makeComponentKey(component *storage.NodeInventory_Components_RHELComponent) string {
	return component.Name + ":" + component.Version + ":" + component.Arch + ":" + component.Module
}
