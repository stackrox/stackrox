package full_nodescan

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

type NodeScanner interface {
	Scan(nodeName string) (*storage.NodeScanV2, error)
}

type NodeScan struct {
}

func (n *NodeScan) Scan(nodeName string) (*storage.NodeScanV2, error) {
	return nil, errors.New("Not implemented")
}
