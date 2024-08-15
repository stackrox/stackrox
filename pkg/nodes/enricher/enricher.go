package enricher

import (
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
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
	// Node Scan / Scanner v2
	EnrichNodeWithInventory(node *storage.Node, nodeInventory *storage.NodeInventory) error
	EnrichNode(node *storage.Node) error
	CreateNodeScanner(integration *storage.NodeIntegration) (types.NodeScannerWithDataSource, error)
	UpsertNodeIntegration(integration *storage.NodeIntegration) error
	RemoveNodeIntegration(id string)
	// Node Index / Scanner v4
	EnrichNodeWithIndexReport(node *storage.Node, indexReport *v4.IndexReport) error
	CreateNodeMatcher(integration *storage.NodeMatcherIntegration) (types.NodeMatcherWithDataSource, error)
	// UpsertNodeMatcherIntegration(integration *storage.NodeMatcherIntegration) error
	// RemoveNodeMatcherIntegration(id string)
}

// CVESuppressor provides enrichment for suppressed CVEs for an node's components.
type CVESuppressor interface {
	EnrichNodeWithSuppressedCVEs(image *storage.Node)
}

// New returns a new NodeEnricher for the given Prometheus metrics subsystem and the Clair node scanner creator.
func New(cves CVESuppressor, subsystem pkgMetrics.Subsystem) NodeEnricher {
	return NewWithCreator(cves, subsystem,
		func() (string, scanners.NodeScannerCreator) {
			return clairify.NodeScannerCreator()
		})
	// func() (string, scanners.NodeScannerCreator) {
	// 	return clairv4.Nod
	// }
	// )
}

// NewWithCreator returns a new NodeEnricher for the given Prometheus metrics subsystem and node scanner creator.
func NewWithCreator(cves CVESuppressor, subsystem pkgMetrics.Subsystem,
	fn func() (string, scanners.NodeScannerCreator),
	// fn4 func() (string, scanners.NodeMatcherCreator),
) NodeEnricher {
	enricher := &enricherImpl{
		cves: cves,

		scanners: make(map[string]types.NodeScannerWithDataSource),
		creators: make(map[string]scanners.NodeScannerCreator),

		matchers:   make(map[string]types.NodeMatcherWithDataSource),
		v4creators: make(map[string]scanners.NodeMatcherCreator),

		metrics: newMetrics(subsystem),
	}
	name, creator := fn()
	enricher.creators[name] = creator

	// v4name, v4creator := fn4()
	// enricher.v4creators[v4name] = v4creator

	return enricher
}
