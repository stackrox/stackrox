package enricher

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/scanners"
	"github.com/stackrox/rox/pkg/scanners/clairify"
	"github.com/stackrox/rox/pkg/scanners/types"
)

var (
	log = logging.LoggerForModule()
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

// New returns a new NodeEnricher instance for the given subsystem.
// (The subsystem is just used for Prometheus metrics.)
func New(cves CVESuppressor, subsystem pkgMetrics.Subsystem) NodeEnricher {
	enricher := &enricherImpl{
		cves: cves,

		scanners: make(map[string]types.NodeScannerWithDataSource),
		creators: make(map[string]scanners.NodeScannerCreator),

		metrics: newMetrics(subsystem),
	}

	clairifyName, clairifyCreator := clairify.NodeScannerCreator()
	enricher.creators[clairifyName] = clairifyCreator

	return enricher
}

// NewWithCreator returns a new NodeEnricher instance for the given subsystem.
// (The subsystem is just used for Prometheus metrics.)
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
