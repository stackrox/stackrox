package full_nodescan

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log             = logging.LoggerForModule()
	_   NodeScanner = (*NodeScan)(nil) // FIXME: Remove
)

type NodeScanner interface {
	Scan(nodeName string) (*sensor.MsgFromCompliance, error)
}

type NodeScan struct {
}

func (n *NodeScan) Scan(nodeName string) (*sensor.MsgFromCompliance, error) {
	return nil, errors.New("Not implemented")
}
