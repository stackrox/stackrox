package enricher

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/logging"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/scanners"
	"github.com/stackrox/rox/pkg/scanners/clairify"
	"github.com/stackrox/rox/pkg/scanners/types"
)

var (
	log = logging.LoggerForModule()
)

// FetchOption determines what attempts should be made to retrieve the metadata
type FetchOption int

// These are all the possible fetch options for the enricher
const (
	UseCachesIfPossible FetchOption = iota
	IgnoreExistingNodes
	ForceRefetch
)

// EnrichmentContext is used to pass options through the enricher without exploding the number of function arguments
type EnrichmentContext struct {
	// FetchOpt define constraints about using external data
	FetchOpt FetchOption
}

// FetchOnlyIfScanEmpty will use the scan that exists in the node unless the fetch opts prohibit it
func (e EnrichmentContext) FetchOnlyIfScanEmpty() bool {
	return e.FetchOpt != IgnoreExistingNodes && e.FetchOpt != ForceRefetch
}

// NodeEnricher provides functions for enriching nodes with vulnerability data.
//go:generate mockgen-wrapper
type NodeEnricher interface {
	EnrichNode(ctx EnrichmentContext, node *storage.Node) error
	CreateNodeScanner(integration *storage.NodeIntegration) (types.NodeScannerWithDataSource, error)
	UpsertNodeIntegration(integration *storage.NodeIntegration) error
	RemoveNodeIntegration(id string)
}

type cveSuppressor interface {
	EnrichNodeWithSuppressedCVEs(node *storage.Node)
}

// New returns a new NodeEnricher instance for the given subsystem.
// (The subsystem is just used for Prometheus metrics.)
func New(cves cveSuppressor, subsystem pkgMetrics.Subsystem, scanCache expiringcache.Cache) NodeEnricher {
	enricher := &enricherImpl{
		cves: cves,

		scanners: make(map[string]types.NodeScannerWithDataSource),
		creators: make(map[string]scanners.NodeScannerCreator),

		scanCache: scanCache,

		metrics: newMetrics(subsystem),
	}

	clairifyName, clairifyCreator := clairify.NodeScannerCreator()
	enricher.creators[clairifyName] = clairifyCreator

	return enricher
}
