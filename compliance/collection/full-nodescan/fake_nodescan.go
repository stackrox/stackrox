package full_nodescan

import (
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
)

var (
	_ NodeScanner = (*FakeNodeScanner)(nil) // FIXME: Remove
)

type FakeNodeScanner struct {
}

func (f *FakeNodeScanner) Scan(nodeName string) (*sensor.FullNodeScan, error) {
	log.Infof("Generating fake scan result message...")
	/*msg := &sensor.MsgFromCompliance{
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
	}*/
	msg := &sensor.FullNodeScan{
		Id:       "",
		Name:     "Fake Testnode",
		ScanTime: &timestamp.Timestamp{Seconds: 42, Nanos: 24},
		Components: &scannerV1.Components{
			Namespace: "Testme OS",
			OsComponents: []*scannerV1.OSComponent{
				{
					Name:      "Test Component",
					Namespace: "tos4",
					Version:   "42.24",
				},
			},
			RhelComponents:     nil,
			LanguageComponents: nil,
		},
		Notes: nil,
	}
	return msg, nil
}
