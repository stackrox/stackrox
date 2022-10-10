package full_nodescan

import (
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
)

var (
	_ NodeScanner = (*FakeNodeScanner)(nil) // FIXME: Remove
)

type FakeNodeScanner struct {
}

func (f *FakeNodeScanner) Scan(nodeName string) (*storage.NodeScanV2, error) {
	log.Infof("Generating fake scan result message...")
	msg := &storage.NodeScanV2{
		NodeId:   "",
		NodeName: "Fake Testnode",
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
