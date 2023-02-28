package nodeinventorizer

import (
	"encoding/json"
	"os"
	"time"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/compliance/collection/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/scanner/database"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stackrox/scanner/pkg/analyzer/nodes"
	"golang.org/x/exp/maps"
)

// CachingScannerOpts control behaviour of all CachingScanner related functions
type CachingScannerOpts struct {
	InventoryCachePath  string              // Path to which a cached inventory is written to
	BackoffFilePath     string              // Path to which the backoff file is written to
	BackoffWaitCallback func(time.Duration) // Callback that gets called if a backoff file is found
}

// CachingScanner is an implementation of NodeInventorizer that keeps a local cache of results.
//
// To reduce strain on the Node, a linear backoff is checked before collecting an inventory.
// Additionally, a cached inventory from an earlier invocation may be used instead of a full inventory run if it is fresh enough.
// Note: This does not prevent strain in case of repeated pod recreation, as both mechanisms are based on an EmptyDir.
type CachingScanner struct {
	InventoryCachePath  string              // Path to which a cached inventory is written to
	BackoffFilePath     string              // Path to which the backoff file is written to
	BackoffWaitCallback func(time.Duration) // Callback that gets called if a backoff file is found
}

// NewCachingScanner returns a ready to use instance of Caching Scanner
func NewCachingScanner(inventoryCachePath string, backoffFilePath string, backoffCallback func(time.Duration)) *CachingScanner {
	return &CachingScanner{opts: &CachingScannerOpts{
		InventoryCachePath:  inventoryCachePath,
		BackoffFilePath:     backoffFilePath,
		BackoffWaitCallback: backoffCallback,
	}}
}

// Scan scans the current node and returns the results as storage.NodeInventory struct
func (c *CachingScanner) Scan(nodeName string) (*storage.NodeInventory, error) {
	// check whether a cached inventory exists that is recent enough to use
	cachedInventory, creationTime := readInventory(c.opts.InventoryCachePath)
	if cachedInventory != nil && isCachedInventoryValid(creationTime) {
		log.Debugf("Using cached node scan created at %v", creationTime)
		return cachedInventory, nil
	}

	// check for existing backoff, wait for specified duration if needed, then persist the new backoff duration
	initialBackoff := env.NodeScanInitialBackoff.DurationSetting()
	currentBackoff := readBackoff(c.opts.BackoffFilePath)

	if currentBackoff > initialBackoff {
		log.Warnf("Found existing node scan backoff file - last scan may have failed. Waiting %v seconds before retrying", currentBackoff.Seconds())
		c.opts.BackoffWaitCallback(currentBackoff)
	}
	writeBackoff(calcNextBackoff(currentBackoff), c.opts.BackoffFilePath)

	// if no inventory exists, or it is too old, collect a fresh one and save it to the cache
	newInventory, err := collectInventory(nodeName)
	if err != nil {
		return nil, err
	}

	if err = writeInventory(newInventory, c.opts.InventoryCachePath); err != nil {
		return nil, errors.Wrap(err, "persisting inventory to cache")
	}

	// Remove backoff directly before returning message, so that a failing/killed container does not lead to
	// frequent rescans of a Node, which are costly and might impact Node performance
	removeBackoff(c.opts.BackoffFilePath)
	return newInventory, nil
}

// readBackoff returns a backoff if found in given file, or the MaxBackoff on any error.
//
// It also checks the loaded backoff to not exceed the configured MaxBackoff.
func readBackoff(path string) time.Duration {
	backoff := env.NodeScanInitialBackoff.DurationSetting()
	maxBackoff := env.NodeScanMaxBackoff.DurationSetting()

	backoffFileContents, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Debug("No node scan backoff file found, continuing without pause")
			return backoff
		}
		log.Warnf("Error reading node scan backoff file, continuing with MaxBackoff of %v. Error: %v", maxBackoff, err)
		return maxBackoff
	}

	// We have an existing backoff counter
	backoff, err = time.ParseDuration(string(backoffFileContents))
	if err != nil {
		log.Warnf("Error reading node scan backoff duration from file, continuing with MaxBackoff of %v. Error: %v", maxBackoff, err)
		return maxBackoff
	}

	// Sanity-check the backoff loaded from disk
	if backoff > maxBackoff {
		log.Warnf("Existing node scan backoff exceeds MaxBackoff. Continuing with MaxBackoff of %v", maxBackoff)
		return maxBackoff
	}

	return backoff
}

func writeBackoff(backoff time.Duration, path string) {
	if err := os.WriteFile(path, []byte(backoff.String()), 0644); err != nil {
		log.Warnf("Error writing node scan backoff file: %v", err)
	}
}

func calcNextBackoff(currentBackoff time.Duration) time.Duration {
	maxBackoff := env.NodeScanMaxBackoff.DurationSetting()
	nextBackoffInterval := currentBackoff + env.NodeScanBackoffIncrement.DurationSetting()
	if nextBackoffInterval > maxBackoff {
		return maxBackoff
	}
	return nextBackoffInterval
}

func removeBackoff(backoffFilePath string) {
	if err := os.Remove(backoffFilePath); err != nil {
		log.Warnf("Could not remove node scan backoff file, subsequent scans may be delayed: %v", err)
	}
}

// inventoryWrap is a private struct that saves a given inventory alongside a creation timestamp which determines its freshness.
type inventoryWrap struct {
	Created   time.Time
	Inventory string
}

// readInventory loads a cached inventory from disk and returns the inventory as well as the creation time.
//
// On any error, readInventory will log the error and return a nil inventory and a time 0.
func readInventory(path string) (inventory *storage.NodeInventory, created time.Time) {
	cacheContents, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Debug("No node scan cache file found, will run a new scan")
			return nil, time.Time{}
		}
		log.Warnf("Unable to read node scan cache, will run a new scan. Error: %v", err)
		return nil, time.Time{}
	}

	// deserialize stored inventory
	var wrap inventoryWrap
	if err := json.Unmarshal(cacheContents, &wrap); err != nil {
		log.Warnf("Unable to load node scan cache contents, will run a new scan. Error: %v", err)
		return nil, time.Time{}
	}

	var cachedInv storage.NodeInventory
	if err := jsonutil.JSONToProto(wrap.Inventory, &cachedInv); err != nil {
		// in this case, also collect a fresh inventory
		log.Warnf("Unable to deserialize node scan cache - will run a new scan. Error: %v", err)
		return nil, time.Time{}
	}

	return &cachedInv, wrap.Created
}

// isCachedInventoryValid returns if a given creation time is considered below the cache threshold.
//
// Additionally, the timestamp is considered invalid if it is based in the future.
func isCachedInventoryValid(created time.Time) bool {
	cacheThreshold := timestamp.TimestampNow().GetSeconds() - int64(env.NodeScanCacheDuration.DurationSetting().Seconds())
	return created.Unix() > cacheThreshold && created.Unix() <= time.Now().Unix()
}

// writeInventory saves a given inventory to disk.
//
// The inventory is wrapped with a creation timestamp to ensure independence of fields in the protobuf.
func writeInventory(inventory *storage.NodeInventory, path string) error {
	inv, err := jsonutil.ProtoToJSON(inventory, jsonutil.OptUnEscape)
	if err != nil {
		return err
	}

	wrap := inventoryWrap{
		Created:   time.Now(),
		Inventory: inv,
	}

	jsonWrap, err := json.Marshal(&wrap)
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, jsonWrap, 0600); err != nil {
		return err
	}
	return nil
}

// collectInventory scans the current node and returns the results as storage.NodeInventory object
func collectInventory(nodeName string) (*storage.NodeInventory, error) {
	metrics.ObserveScansTotal(nodeName)
	startTime := time.Now()

	// uncertifiedRHEL is set to false, as scans are only supported on RHCOS for now,
	// which only exists in certified versions
	componentsHost, err := nodes.Analyze(nodeName, "/host/", nodes.AnalyzeOpts{UncertifiedRHEL: false, IsRHCOSRequired: true})

	scanDuration := time.Since(startTime)
	metrics.ObserveScanDuration(scanDuration, nodeName, err)
	log.Debugf("Collecting Node Inventory took %f seconds", scanDuration.Seconds())

	if err != nil {
		log.Errorf("Error scanning node /host inventory: %v", err)
		return nil, err
	}
	log.Debugf("Components found under /host: %v", componentsHost)

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
		NodeId:     uuid.Nil.String(), // The NodeID is not available in compliance, but only on Sensor and later on
		NodeName:   nodeName,
		ScanTime:   timestamp.TimestampNow(),
		Components: protoComponents,
		Notes:      []storage.NodeInventory_Note{storage.NodeInventory_LANGUAGE_CVES_UNAVAILABLE},
	}

	metrics.ObserveNodeInventoryScan(m)
	return m, nil
}

// TODO(ROX-14029): Move conversion function into Sensor
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

// TODO(ROX-14029): Move conversion function into Sensor
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

// TODO(ROX-14029): Move conversion function into Sensor
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
