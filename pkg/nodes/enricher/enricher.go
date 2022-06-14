package enricher

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/logging"
	pkgMetrics "github.com/stackrox/stackrox/pkg/metrics"
	"github.com/stackrox/stackrox/pkg/scanners"
	"github.com/stackrox/stackrox/pkg/scanners/clairify"
	"github.com/stackrox/stackrox/pkg/scanners/types"
)

var (
	log = logging.LoggerForModule()
)

// NodeEnricher provides functions for enriching nodes with vulnerability data.
//go:generate mockgen-wrapper
type NodeEnricher interface {
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
