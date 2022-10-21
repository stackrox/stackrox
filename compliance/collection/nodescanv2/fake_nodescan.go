package nodescanv2

import (
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
)

// FakeNodeScanner can be used to send fake messages that would be emitted by NodeScanV2
type FakeNodeScanner struct {
}

// Scan returns a fake message in the same format a real NodeScanV2 would produce
func (f *FakeNodeScanner) Scan(nodeName string) (*storage.NodeScanV2, error) {
	log.Infof("Generating fake scan result message...")
	msg := &storage.NodeScanV2{
		NodeId:   "",
		NodeName: nodeName,
		ScanTime: timestamp.TimestampNow(),
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
