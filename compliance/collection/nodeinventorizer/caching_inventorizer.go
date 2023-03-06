package nodeinventorizer

import (
	"encoding/json"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/jsonutil"
)

const (
	backoffMultiplier = 2
)

// CachingScanner is an implementation of NodeInventorizer that keeps a local cache of results.
//
// To reduce strain on the Node, an exponential backoff is checked before collecting an inventory.
// Additionally, a cached inventory from an earlier invocation may be used instead of a full inventory run if it is fresh enough.
// Note: This does not prevent strain in case of repeated pod recreation, as both mechanisms are based on an EmptyDir.
type CachingScanner struct {
	analyzer            NodeInventorizer
	inventoryCachePath  string              // Path to which a cached inventory is written to
	cacheDuration       time.Duration       // Duration for which a cached inventory will be considered new enough
	initialBackoff      time.Duration       // First backoff interval the node scan starts with
	maxBackoff          time.Duration       // Maximum duration that the backoff is allowed to grow to
	backoffWaitCallback func(time.Duration) // Callback that gets called if a backoff file is found
}

// inventoryWrap is a private struct that saves a given inventory alongside some meta-information.
type inventoryWrap struct {
	CacheValidUntil      time.Time     // CacheValidUntil indicates whether the cached inventory is fresh enough to use.
	RetryBackoffDuration time.Duration // RetryBackoffDuration contains the duration a scan waits before its next iteration.
	CachedInventory      string        // serialized form of the cached inventory
}

// NewCachingScanner returns a ready to use instance of Caching Scanner
func NewCachingScanner(analyzer NodeInventorizer, inventoryCachePath string, cacheDuration time.Duration, initialBackoff time.Duration, maxBackoff time.Duration, backoffCallback func(time.Duration)) *CachingScanner {
	return &CachingScanner{
		analyzer:            analyzer,
		inventoryCachePath:  inventoryCachePath,
		cacheDuration:       cacheDuration,
		initialBackoff:      initialBackoff,
		maxBackoff:          maxBackoff,
		backoffWaitCallback: backoffCallback,
	}
}

// Scan scans the current node and returns the results as storage.NodeInventory struct.
// A cached version is returned if it exists and is fresh enough.
// Otherwise, a new scan guarded by a backoff is run by the injected analyzer.
func (c *CachingScanner) Scan(nodeName string) (*storage.NodeInventory, error) {
	// check whether a cached inventory exists that has not exceeded its validity
	cachedInv, validUntil := readCachedInventory(c.inventoryCachePath)
	if cachedInv != nil && !validUntil.IsZero() && validUntil.After(time.Now()) {
		log.Debugf("Using cached node scan (valid until %v)", validUntil)
		return cachedInv, nil
	}

	// check for existing backoff, wait for specified duration if needed, then persist the new backoff duration
	backoffDuration := c.initialBackoff
	cachedBackoff := readBackoff(c.inventoryCachePath)
	if cachedBackoff > 0 {
		backoffDuration = min(cachedBackoff, c.maxBackoff)
		if backoffDuration > 0 {
			log.Warnf("Found existing node scan backoff - last scan may have failed. Waiting %v seconds before retrying", backoffDuration.Seconds())
			c.backoffWaitCallback(backoffDuration)
			backoffDuration = c.calcNextBackoff(backoffDuration) // Set the next backoff duration to persist.
		}
	}

	// Write backoff duration to cache
	if err := writeBackoff(backoffDuration, c.inventoryCachePath); err != nil {
		return nil, errors.Wrap(err, "writing node scan backoff file")
	}

	// if no inventory exists, or it is too old, collect a fresh one
	newInventory, err := c.analyzer.Scan(nodeName)
	if err != nil {
		return nil, err
	}

	// Write inventory to cache
	if err := writeCachedInventory(newInventory, time.Now().Add(c.cacheDuration), c.inventoryCachePath); err != nil {
		return nil, errors.Wrap(err, "persisting inventory to cache")
	}

	return newInventory, nil
}

func (c *CachingScanner) calcNextBackoff(currentBackoff time.Duration) time.Duration {
	return min(currentBackoff*backoffMultiplier, c.maxBackoff)
}

func min(d1 time.Duration, d2 time.Duration) time.Duration {
	if d1 > d2 {
		return d2
	}
	return d1
}

func readBackoff(path string) time.Duration {
	wrap := readInventoryWrap(path)
	if wrap != nil {
		return wrap.RetryBackoffDuration
	}
	return 0
}

func writeBackoff(backoff time.Duration, path string) error {
	wrap := inventoryWrap{
		CacheValidUntil:      time.Time{},
		RetryBackoffDuration: backoff,
		CachedInventory:      "",
	}
	return writeInventoryWrap(wrap, path)
}

func readCachedInventory(path string) (inventory *storage.NodeInventory, validUntil time.Time) {
	wrap := readInventoryWrap(path)
	if wrap == nil || wrap.CachedInventory == "" {
		return nil, time.Time{}
	}

	var cachedInv storage.NodeInventory
	if err := jsonutil.JSONToProto(wrap.CachedInventory, &cachedInv); err != nil {
		log.Warnf("error unmarshalling node scan from cache: %v", err)
		return nil, time.Time{}
	}
	return &cachedInv, wrap.CacheValidUntil
}

func writeCachedInventory(inventory *storage.NodeInventory, validUntil time.Time, path string) error {
	strInv, err := jsonutil.ProtoToJSON(inventory)
	if err != nil {
		return err
	}

	wrap := inventoryWrap{
		CacheValidUntil:      validUntil,
		RetryBackoffDuration: 0,
		CachedInventory:      strInv,
	}
	return writeInventoryWrap(wrap, path)
}

func readInventoryWrap(path string) *inventoryWrap {
	cacheContents, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Debug("No node scan cache file found")
			return nil
		}
		log.Warnf("Unable to read node scan cache. Error: %v", err)
		return nil
	}

	// deserialize stored inventory
	var wrap inventoryWrap
	if err := json.Unmarshal(cacheContents, &wrap); err != nil {
		log.Warnf("Unable to load node scan cache contents. Error: %v", err)
		return nil
	}
	return &wrap
}

func writeInventoryWrap(w inventoryWrap, path string) error {
	jsonWrap, err := json.Marshal(&w)
	if err != nil {
		return err
	}

	return os.WriteFile(path, jsonWrap, 0600)
}
