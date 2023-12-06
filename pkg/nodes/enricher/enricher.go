package enricher

import (
	"github.com/stackrox/rox/generated/storage"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/scanners"
	"github.com/stackrox/rox/pkg/scanners/clairify"
	"github.com/stackrox/rox/pkg/scanners/types"
)

// NodeEnricher provides functions for enriching nodes with vulnerability data.
//
//go:generate mockgen-wrapper
type NodeEnricher interface {
	EnrichNodeWithInventory(node *storage.Node, nodeInventory *storage.NodeInventory) error
	EnrichNode(node *storage.Node) error
	CreateNodeScanner(integration *storage.NodeIntegration) (types.NodeScannerWithDataSource, error)
	UpsertNodeIntegration(integration *storage.NodeIntegration) error
	RemoveNodeIntegration(id string)
}

// CVESuppressor provides enrichment for suppressed CVEs for an node's components.
type CVESuppressor interface {
	EnrichNodeWithSuppressedCVEs(image *storage.Node)
}

// New returns a new NodeEnricher for the given Prometheus metrics subsystem and the Clair node scanner creator.
func New(cves CVESuppressor, subsystem pkgMetrics.Subsystem) NodeEnricher {
	return NewWithCreator(cves, subsystem, func() (string, scanners.NodeScannerCreator) {
		return clairify.NodeScannerCreator()
	})
}

// NewWithCreator returns a new NodeEnricher for the given Prometheus metrics subsystem and node scanner creator.
func NewWithCreator(cves CVESuppressor, subsystem pkgMetrics.Subsystem, fn func() (string, scanners.NodeScannerCreator)) NodeEnricher {
	enricher := &enricherImpl{
		cves: cves,

		scanners: make(map[string]types.NodeScannerWithDataSource),
		creators: make(map[string]scanners.NodeScannerCreator),

		metrics: newMetrics(subsystem),
	}
	name, creator := fn()
	enricher.creators[name] = creator

	return enricher
}
