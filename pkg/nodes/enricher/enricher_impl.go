package enricher

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/nodes/converter"
	pkgScanners "github.com/stackrox/rox/pkg/scanners"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/sync"
)

type enricherImpl struct {
	cves CVESuppressor

	lock     sync.RWMutex
	scanners map[string]types.NodeScannerWithDataSource

	creators map[string]pkgScanners.NodeScannerCreator

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
func (e *enricherImpl) EnrichNode(node *storage.Node) error {
	// Clear any pre-existing notes, as it will all be filled here.
	// Note: this is valid even if node.Notes is nil.
	node.Notes = node.Notes[:0]

	err := e.enrichWithScan(node)
	if err != nil {
		node.Notes = append(node.Notes, storage.Node_MISSING_SCAN_DATA)
	}

	e.cves.EnrichNodeWithSuppressedCVEs(node)

	return err
}

func (e *enricherImpl) enrichWithScan(node *storage.Node) error {
	errorList := errorhelpers.NewErrorList(fmt.Sprintf("error scanning node %s:%s", node.GetClusterName(), node.GetName()))

	e.lock.RLock()
	scanners := make([]types.NodeScannerWithDataSource, 0, len(e.scanners))
	for _, scanner := range e.scanners {
		scanners = append(scanners, scanner)
	}
	e.lock.RUnlock()

	if len(scanners) == 0 {
		errorList.AddError(errors.New("no node scanners are integrated"))
		return errorList.ToError()
	}

	for _, scanner := range scanners {
		if err := e.enrichNodeWithScanner(node, scanner.GetNodeScanner()); err != nil {
			errorList.AddError(err)
			continue
		}

		return nil
	}

	return errorList.ToError()
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
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		converter.FillV2NodeVulnerabilities(node)
		for _, component := range node.GetScan().GetComponents() {
			component.Vulns = nil
		}
	}
	FillScanStats(node)

	return nil
}

// FillScanStats fills in the higher level stats from the scan data.
func FillScanStats(n *storage.Node) {
	if n.GetScan() == nil {
		return
	}

	n.SetComponents = &storage.Node_Components{
		Components: int32(len(n.GetScan().GetComponents())),
	}

	var fixedByProvided bool
	var nodeTopCVSS float32
	vulns := make(map[string]bool)
	for _, c := range n.GetScan().GetComponents() {
		var componentTopCVSS float32
		var hasVulns bool

		if env.PostgresDatastoreEnabled.BooleanSetting() {
			for _, v := range c.GetVulnerabilities() {
				hasVulns = true
				if _, ok := vulns[v.GetCveBaseInfo().GetCve()]; !ok {
					vulns[v.GetCveBaseInfo().GetCve()] = false
				}

				if v.GetCvss() > componentTopCVSS {
					componentTopCVSS = v.GetCvss()
				}

				if v.GetSetFixedBy() == nil {
					continue
				}

				fixedByProvided = true
				if v.GetFixedBy() != "" {
					vulns[v.GetCveBaseInfo().GetCve()] = true
				}
			}
		} else {
			for _, v := range c.GetVulns() {
				hasVulns = true
				if _, ok := vulns[v.GetCve()]; !ok {
					vulns[v.GetCve()] = false
				}

				if v.GetCvss() > componentTopCVSS {
					componentTopCVSS = v.GetCvss()
				}

				if v.GetSetFixedBy() == nil {
					continue
				}

				fixedByProvided = true
				if v.GetFixedBy() != "" {
					vulns[v.GetCve()] = true
				}
			}
		}

		if hasVulns {
			c.SetTopCvss = &storage.EmbeddedNodeScanComponent_TopCvss{
				TopCvss: componentTopCVSS,
			}
		}

		if componentTopCVSS > nodeTopCVSS {
			nodeTopCVSS = componentTopCVSS
		}
	}

	n.SetCves = &storage.Node_Cves{
		Cves: int32(len(vulns)),
	}

	if len(vulns) > 0 {
		n.SetTopCvss = &storage.Node_TopCvss{
			TopCvss: nodeTopCVSS,
		}
	}

	if int32(len(vulns)) == 0 || fixedByProvided {
		var numFixableVulns int32
		for _, fixable := range vulns {
			if fixable {
				numFixableVulns++
			}
		}
		n.SetFixable = &storage.Node_FixableCves{
			FixableCves: numFixableVulns,
		}
	}
}
