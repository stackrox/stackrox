package enricher

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/scanners"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/sync"
)

type enricherImpl struct {
	cves cveSuppressor

	lock     sync.Mutex
	scanners map[string]types.NodeScannerWithDataSource

	creators map[string]scanners.NodeScannerCreator

	scanCache expiringcache.Cache

	metrics metrics
}

// UpsertNodeIntegration creates or updates a node integration.
func (e *enricherImpl) UpsertNodeIntegration(integration *storage.NodeIntegration) error {
	scanner, err := e.CreateNodeScanner(integration)
	if err != nil {
		return err
	}

	e.lock.Lock()
	defer e.lock.Unlock()

	e.scanners[integration.GetId()] = scanner

	return nil
}

// RemoveNodeIntegration deletes a node integration with the given id if it exists.
func (e *enricherImpl) RemoveNodeIntegration(id string) {
	e.lock.Lock()
	defer e.lock.Unlock()

	delete(e.scanners, id)
}

// EnrichNode enriches a node with the integration set present.
func (e *enricherImpl) EnrichNode(ctx EnrichmentContext, node *storage.Node) error {
	err := e.enrichWithScan(ctx, node)

	e.cves.EnrichNodeWithSuppressedCVEs(node)

	return err
}

func (e *enricherImpl) enrichWithScan(ctx EnrichmentContext, node *storage.Node) error {
	// Attempt to short-circuit before checking scanners.
	if ctx.FetchOnlyIfScanEmpty() && node.Scan != nil {
		return nil
	}
	if e.populateFromCache(ctx, node) {
		return nil
	}

	errorList := errorhelpers.NewErrorList(fmt.Sprintf("error scanning node %s:%s", node.GetClusterName(), node.GetName()))
	if len(e.scanners) == 0 {
		errorList.AddError(errors.New("no node scanners are integrated"))
		return errorList.ToError()
	}

	for _, scanner := range e.scanners {
		if err := e.enrichNodeWithScanner(node, scanner); err != nil {
			errorList.AddError(err)
			continue
		}

		return nil
	}

	return errorList.ToError()
}

func (e *enricherImpl) populateFromCache(ctx EnrichmentContext, node *storage.Node) bool {
	if ctx.FetchOpt == ForceRefetch {
		return false
	}
	scanValue := e.scanCache.Get(node.GetId())
	if scanValue == nil {
		e.metrics.IncrementScanCacheMiss()
		return false
	}

	e.metrics.IncrementScanCacheHit()
	node.Scan = scanValue.(*storage.NodeScan).Clone()
	return true
}

func (e *enricherImpl) enrichNodeWithScanner(node *storage.Node, scanner types.NodeScanner) error {
	sema := scanner.MaxConcurrentNodeScanSemaphore()
	_ = sema.Acquire(context.Background(), 1)
	defer sema.Release(1)

	scanStartTime := time.Now()
	scan, err := scanner.GetNodeScan(node)
	e.metrics.SetScanDurationTime(scanStartTime, scanner.Name(), err)
	if err != nil {
		return errors.Wrapf(err, "Error scanning '%s:%s' with scanner %q", node.GetClusterName(), node.GetName(), scanner.Name())
	}
	if scan == nil {
		return nil
	}

	node.Scan = scan

	// Clone the cachedScan because the scan is used within the node leading to race conditions
	cachedScan := scan.Clone()
	e.scanCache.Add(node.GetId(), cachedScan)

	return nil
}
