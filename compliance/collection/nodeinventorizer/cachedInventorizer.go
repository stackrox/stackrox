package nodeinventorizer

import (
	"os"
	"time"

	"github.com/gogo/protobuf/proto"
	timestamp "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	cmetrics "github.com/stackrox/rox/compliance/collection/metrics"
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
	// check for existing backoff, wait for specified duration if needed, then persist the new backoff duration
	initialBackoff := env.NodeInventoryInitialBackoff.DurationSetting()
	currentBackoff, err := getCurrentBackoff(opts)
	if err != nil {
		return nil, err
	}
	if *currentBackoff > initialBackoff {
		log.Warnf("Found existing backoff - last scan may have failed. Waiting %v seconds before running next inventory", currentBackoff)
		opts.BackoffWaitCallback(*currentBackoff)
	}
	nextBackoff := calcNextBackoff(*currentBackoff)
	bErr := writeBackoff(nextBackoff, opts.BackoffFilePath)
	if bErr != nil {
		log.Warnf("Error while persisting inventory backoff interval: %v", bErr)
	}

	// check whether a cached inventory exists that is recent enough to use
	cachedInventory := loadCachedInventory(opts)
	if cachedInventory != nil && isInventoryInCacheDuration(cachedInventory) {
		log.Debugf("Using cached scan created at %v", cachedInventory.GetScanTime())
		return createAndObserveMessage(cachedInventory), nil
	}

	// if no inventory exists, or it is too old, collect a fresh one and save it to the cache
	newInventory, err := opts.Scanner.Scan(opts.NodeName)
	if err != nil {
		return nil, err
	}

	err = persistInventoryToCache(newInventory, opts)
	if err != nil {
		return nil, err
	}

	removeBackoff(opts.BackoffFilePath) // Remove backoff file as late as possible
	return createAndObserveMessage(newInventory), nil
}

func getCurrentBackoff(opts *InventoryScanOpts) (*time.Duration, error) {
	backoffInterval := env.NodeInventoryInitialBackoff.DurationSetting()

	backoffFileContents, err := os.ReadFile(opts.BackoffFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Debug("No backoff found, continuing without pause")
		} else {
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

func writeBackoff(backoff time.Duration, path string) error {
	err := os.WriteFile(path, []byte(backoff.String()), 0600)
	if err != nil {
		return err
	}
	return nil
}

func calcNextBackoff(currentBackoff time.Duration) time.Duration {
	maxBackoff := env.NodeInventoryMaxBackoff.DurationSetting()
	nextBackoffInterval := currentBackoff + env.NodeInventoryBackoffIncrement.DurationSetting()
	if nextBackoffInterval > maxBackoff {
		log.Debugf("Backoff interval hit upper boundary. Cutting from %v to %v", nextBackoffInterval, maxBackoff)
		nextBackoffInterval = maxBackoff
	}
	return nextBackoffInterval
}

func removeBackoff(backoffFilePath string) {
	if err := os.Remove(backoffFilePath); err != nil {
		log.Warnf("Could not remove scan backoff state file: %v", err)
	}
}

func loadCachedInventory(opts *InventoryScanOpts) *storage.NodeInventory {
	var cachedInv *storage.NodeInventory

	cacheContents, err := os.ReadFile(opts.InventoryCachePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Debug("No cache file found, running new inventory")
		}
		log.Warnf("Unable to read inventory cache, running new inventory. Error: %v", err)
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

func isInventoryInCacheDuration(inventory *storage.NodeInventory) bool {
	scanTime := inventory.GetScanTime()
	cacheThreshold := timestamp.TimestampNow().GetSeconds() - int64(env.NodeInventoryCacheDuration.DurationSetting().Seconds())

	if scanTime != nil && scanTime.GetSeconds() > cacheThreshold {
		return true
	}
	return false
}

func createAndObserveMessage(inventory *storage.NodeInventory) *sensor.MsgFromCompliance {
	msg := &sensor.MsgFromCompliance{
		Node: inventory.GetNodeName(),
		Msg:  &sensor.MsgFromCompliance_NodeInventory{NodeInventory: inventory},
	}
	cmetrics.ObserveInventoryProtobufMessage(msg)
	return msg
}

func persistInventoryToCache(inventory *storage.NodeInventory, opts *InventoryScanOpts) error {
	inv, err := proto.Marshal(inventory)
	if err != nil {
		return err
	}
	if err := os.WriteFile(opts.InventoryCachePath, inv, 0600); err != nil {
		return err
	}
	return nil
}
