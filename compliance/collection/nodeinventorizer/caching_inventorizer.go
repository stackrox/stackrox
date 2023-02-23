package nodeinventorizer

import (
	"os"
	"time"

	"github.com/gogo/protobuf/proto"
	timestamp "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
)

// InventoryScanOpts control behaviour of all node inventory related functions
type InventoryScanOpts struct {
	NodeName            string
	Scanner             NodeInventorizer
	InventoryCachePath  string
	BackoffFilePath     string
	BackoffWaitCallback func(time.Duration)
}

// TriggerNodeInventory creates a Node Inventory to send to Sensor.
// To reduce strain on the Node, a linear backoff is checked before collecting an inventory.
// Additionally, a cached inventory from an earlier invocation may be used instead of a full inventory run if it is fresh enough.
// Note: This does not prevent strain in case of repeated pod recreation, as both mechanisms are based on an EmptyDir.
func TriggerNodeInventory(opts *InventoryScanOpts) (*sensor.MsgFromCompliance, error) {
	// check whether a cached inventory exists that is recent enough to use
	cachedInventory := loadCachedInventory(opts)
	if cachedInventory != nil && isCachedInventoryValid(cachedInventory) {
		log.Debugf("Using cached node scan created at %v", cachedInventory.GetScanTime())
		return createMessage(cachedInventory), nil
	}

	// check for existing backoff, wait for specified duration if needed, then persist the new backoff duration
	initialBackoff := env.NodeScanInitialBackoff.DurationSetting()
	currentBackoff, err := getCurrentBackoff(opts.BackoffFilePath)
	if err != nil {
		// Set to maxBackoff to make sure scanning still happens, even if the backoff file gets corrupted
		*currentBackoff = env.NodeScanMaxBackoff.DurationSetting()
	}
	if *currentBackoff > initialBackoff {
		log.Warnf("Found existing node scan backoff file - last scan may have failed. Waiting %v seconds before retrying", currentBackoff.Seconds())
		opts.BackoffWaitCallback(*currentBackoff)
	}
	writeBackoff(calcNextBackoff(*currentBackoff), opts.BackoffFilePath)

	// if no inventory exists, or it is too old, collect a fresh one and save it to the cache
	newInventory, err := opts.Scanner.Scan(opts.NodeName)
	if err != nil {
		return nil, err
	}

	err = persistInventoryToCache(newInventory, opts)
	if err != nil {
		return nil, errors.Wrap(err, "persisting inventory to cache")
	}

	// Remove backoff directly before returning message, so that a failing/killed container does not lead to
	// frequent rescans of a Node, which are costly and might impact Node performance
	removeBackoff(opts.BackoffFilePath)
	return createMessage(newInventory), nil
}

func getCurrentBackoff(backoffFilePath string) (*time.Duration, error) {
	backoffInterval := env.NodeScanInitialBackoff.DurationSetting()

	backoffFileContents, err := os.ReadFile(backoffFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Debug("No backoff found, continuing without pause")
		} else {
			// return
			return nil, err
		}
	} else {
		// We have an existing backoff counter
		backoffInterval, err = time.ParseDuration(string(backoffFileContents))
		if err != nil {
			return nil, err
		}
	}

	return &backoffInterval, nil
}

func writeBackoff(backoff time.Duration, path string) {
	err := os.WriteFile(path, []byte(backoff.String()), 0600)
	if err != nil {
		log.Warnf("Error writing backoff marker: %v", err)
	}
}

func calcNextBackoff(currentBackoff time.Duration) time.Duration {
	maxBackoff := env.NodeScanMaxBackoff.DurationSetting()
	nextBackoffInterval := currentBackoff + env.NodeScanBackoffIncrement.DurationSetting()
	if nextBackoffInterval > maxBackoff {
		log.Debugf("Backoff interval hit upper boundary. Cutting from %v to %v", nextBackoffInterval, maxBackoff)
		nextBackoffInterval = maxBackoff
	}
	return nextBackoffInterval
}

func removeBackoff(backoffFilePath string) {
	if err := os.Remove(backoffFilePath); err != nil {
		log.Warnf("Could not remove backoff marker, subsequent scans may be delayed: %v", err)
	}
}

func loadCachedInventory(opts *InventoryScanOpts) *storage.NodeInventory {
	var cachedInv *storage.NodeInventory

	cacheContents, err := os.ReadFile(opts.InventoryCachePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Debug("No cache file found, running new inventory")
		} else {
			log.Warnf("Unable to read inventory cache, running new inventory. Error: %v", err)
		}
	} else {
		// deserialize stored inventory into
		cachedInv = &storage.NodeInventory{}
		if e := proto.Unmarshal(cacheContents, cachedInv); e != nil {
			// in this case, also collect a fresh inventory
			log.Warnf("Unable to deserialize inventory cache - running new inventory. Error: %v", e)
			return nil
		}
	}
	return cachedInv
}

func isCachedInventoryValid(inventory *storage.NodeInventory) bool {
	scanTime := inventory.GetScanTime()
	cacheThreshold := timestamp.TimestampNow().GetSeconds() - int64(env.NodeScanCacheDuration.DurationSetting().Seconds())

	return scanTime != nil && scanTime.GetSeconds() > cacheThreshold
}

func createMessage(inventory *storage.NodeInventory) *sensor.MsgFromCompliance {
	msg := &sensor.MsgFromCompliance{
		Node: inventory.GetNodeName(),
		Msg:  &sensor.MsgFromCompliance_NodeInventory{NodeInventory: inventory},
	}
	return msg
}

func persistInventoryToCache(inventory *storage.NodeInventory, opts *InventoryScanOpts) error {
	inv, err := proto.Marshal(inventory)
	if err != nil {
		return err
	}
	if err := os.WriteFile(opts.InventoryCachePath, inv, 0644); err != nil {
		return err
	}
	return nil
}
