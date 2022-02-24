package enricher

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scanners/types"
)

var _ types.NodeScannerWithDataSource = (*nodeScannerWithDataSource)(nil)

type nodeScannerWithDataSource struct {
	nodeScanner types.NodeScanner
	datasource  *storage.DataSource
}

func (n *nodeScannerWithDataSource) GetNodeScanner() types.NodeScanner {
	return n.nodeScanner
}

func (n *nodeScannerWithDataSource) DataSource() *storage.DataSource {
	return n.datasource
}

// CreateNodeScanner creates a types.NodeScannerWithDataSource out of the given storage.NodeIntegration.
func (e *enricherImpl) CreateNodeScanner(source *storage.NodeIntegration) (types.NodeScannerWithDataSource, error) {
	creator, exists := e.creators[source.GetType()]
	if !exists {
		return nil, fmt.Errorf("scanner with type %q does not exist", source.GetType())
	}
	scanner, err := creator(source)
	if err != nil {
		return nil, err
	}
	return &nodeScannerWithDataSource{
		nodeScanner: scanner,
		datasource: &storage.DataSource{
			Id:   source.GetId(),
			Name: source.GetName(),
		},
	}, nil
}
