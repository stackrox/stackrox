package index

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/compliance/node"
	"github.com/stackrox/rox/compliance/utils"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
)

var (
	_ node.NodeIndexer = (*cachingNodeIndexer)(nil)
)

type cachingNodeIndexer struct {
	indexerConfig NodeIndexerConfig
	indexer       node.NodeIndexer
	cacheDuration time.Duration
	cachePath     string
}

// reportWrap is an internal representation of a generated IndexReport with additional metadata
type reportWrap struct {
	CacheValidUntil time.Time       // CacheValidUntil indicates the cutoff until the report is considered fresh enough to use.
	Report          *v4.IndexReport // Report contains the created IndexReport to cache.
}

// NewCachingNodeIndexer creates a new cached node indexer
func NewCachingNodeIndexer(indexerConfig NodeIndexerConfig, cacheDuration time.Duration, cachePath string) node.NodeIndexer {
	log.Debugf("Creating new cached Node Indexer with cache duration %s and cache path %s",
		cacheDuration, cachePath)
	indexer := NewNodeIndexer(indexerConfig)
	return &cachingNodeIndexer{indexer: indexer, indexerConfig: indexerConfig, cacheDuration: cacheDuration, cachePath: cachePath}
}

// IndexNode will serve a cached report if it's fresh enough. Otherwise, it creates a new one and caches it.
func (c cachingNodeIndexer) IndexNode(ctx context.Context) (*v4.IndexReport, error) {
	r, err := loadCachedReport(c.cachePath)
	if err != nil {
		log.Debugf("Unable to load cached report. Will create a new cache. Error: %v. ", err)
	}
	if r != nil {
		return r, nil
	}

	// Cache is too old - create and cache new index report
	report, err := c.indexer.IndexNode(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error indexing node")
	}
	log.Debugf("Caching report for %s at path %s", c.cacheDuration, c.cachePath)
	err = cacheReport(report, c.cachePath, c.cacheDuration)
	if err != nil {
		log.Warnf("Failed to cache report - caching will be ineffective: %v", err)
	}
	return report, nil
}

func cacheReport(report *v4.IndexReport, cachePath string, cacheDuration time.Duration) error {
	wrap := &reportWrap{
		Report:          report,
		CacheValidUntil: time.Now().Add(cacheDuration),
	}
	jsonWrap, err := json.Marshal(wrap)
	if err != nil {
		return err
	}
	return os.WriteFile(cachePath, jsonWrap, 0600)
}

// loadCachedReport returns a report if found and new enough.
func loadCachedReport(cachePath string) (*v4.IndexReport, error) {
	wrap, err := loadCachedWrap(cachePath)
	if err != nil {
		return nil, err
	}
	if wrap != nil && wrap.CacheValidUntil.After(time.Now()) {
		return wrap.Report, nil
	}
	return nil, errors.New("cached report too old")
}

func loadCachedWrap(cachePath string) (*reportWrap, error) {
	rawWrap, err := os.ReadFile(cachePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errors.Wrap(err, "No cached index report wrap found")
		}
		return nil, errors.Wrap(err, "reading cached index report wrap")
	}

	var wrap reportWrap
	if err := json.Unmarshal(rawWrap, &wrap); err != nil {
		return nil, errors.Wrap(err, "unmarshalling cached index report wrap")
	}
	return &wrap, nil
}

func (c cachingNodeIndexer) GetIntervals() *utils.NodeScanIntervals {
	return c.indexer.GetIntervals()
}
