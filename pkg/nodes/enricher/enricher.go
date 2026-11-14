package enricher

import (
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/scanners"
	"github.com/stackrox/rox/pkg/scanners/clairify"
	"github.com/stackrox/rox/pkg/scanners/scannerv4"
	"github.com/stackrox/rox/pkg/scanners/types"
)

// NodeEnricher provides functions for enriching nodes with vulnerability data.
//
//go:generate mockgen-wrapper
type NodeEnricher interface {
	EnrichNodeWithVulnerabilities(node *storage.Node, nodeInventory *storage.NodeInventory, indexReport *v4.IndexReport) error
	EnrichNode(node *storage.Node) error
	CreateNodeScanner(integration *storage.NodeIntegration) (types.NodeScannerWithDataSource, error)
	UpsertNodeIntegration(integration *storage.NodeIntegration) error
	RemoveNodeIntegration(id string)
}

// CVESuppressor provides enrichment for suppressed CVEs for an node's components.
type CVESuppressor interface {
	EnrichNodeWithSuppressedCVEs(image *storage.Node)
}

// New returns a new NodeEnricher for the given Prometheus metrics subsystem and the node scanner creators.
func New(cves CVESuppressor, subsystem pkgMetrics.Subsystem) NodeEnricher {
	// The legacy StackRox Scanner (Scanner V2 / Clairify) is removed in 5.0 and is
	// only registered when it is explicitly enabled for backward compatibility,
	// mirroring the image scanner factory (pkg/scanners/factory.go). Scanner V4 is
	// always registered.
	var legacyCreator func() (string, scanners.NodeScannerCreator)
	if features.LegacyScanner.Enabled() {
		legacyCreator = func() (string, scanners.NodeScannerCreator) {
			return clairify.NodeScannerCreator()
		}
	}
	return NewWithCreator(cves, subsystem,
		legacyCreator,
		func() (string, scanners.NodeScannerCreator) {
			return scannerv4.NodeScannerCreator()
		})
}

// NewWithCreator returns a new NodeEnricher for the given Prometheus metrics subsystem and node scanner creator.
func NewWithCreator(cves CVESuppressor, subsystem pkgMetrics.Subsystem,
	fn func() (string, scanners.NodeScannerCreator),
	fn4 func() (string, scanners.NodeScannerCreator),
) NodeEnricher {
	enricher := &enricherImpl{
		cves: cves,

		scanners: make(map[string]types.NodeScannerWithDataSource),
		creators: make(map[string]scanners.NodeScannerCreator),

		metrics: newMetrics(subsystem),
	}
	// fn (the legacy creator) is optional: New passes nil when the legacy scanner
	// is disabled so no clairify node-scanner creator is registered.
	if fn != nil {
		name, creator := fn()
		enricher.creators[name] = creator
	}
	name4, creator4 := fn4()
	enricher.creators[name4] = creator4

	return enricher
}
