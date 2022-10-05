package full_nodescan

import (
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
)

var (
	_ NodeScanner = (*FakeNodeScanner)(nil) // FIXME: Remove
)

type FakeNodeScanner struct {
}

func (f *FakeNodeScanner) Scan(nodeName string) (*sensor.MsgFromCompliance, error) {
	log.Infof("Generating fake scan result message...")
	msg := &sensor.MsgFromCompliance{
		Node: nodeName,
		Msg: &sensor.MsgFromCompliance_NodeScan{
			NodeScan: &storage.NodeScan{
				OperatingSystem: "Fake RHEL",
				Components: []*storage.EmbeddedNodeScanComponent{
					{
						Name:    "Fake Component",
						Version: "4.2",
					},
				},
			},
		},
	}
	return msg, nil
}
