package nodescanv2

import (
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
)

var (
	log = logging.LoggerForModule()
)

// FakeNodeScanner can be used to send fake messages that would be emitted by NodeInventory
type FakeNodeScanner struct {
}

// Scan returns a fake message in the same format a real NodeInventory would produce
func (f *FakeNodeScanner) Scan(nodeName string) (*storage.NodeInventory, error) {
	log.Infof("Generating fake scan result message...")
	msg := &storage.NodeInventory{
		NodeId:   "",
		NodeName: nodeName,
		ScanTime: timestamp.TimestampNow(),
		Components: &scannerV1.Components{
			Namespace: "Testme OS",
			RhelComponents: []*scannerV1.RHELComponent{
				{
					Name:      "vim-minimal",
					Namespace: "rhel:8",
					Version:   "2:7.4.629-6.el8.x86_64",
					Arch:      "x86_64",
					Module:    "FakeMod",
					Cpes:      []string{"cpe:/a:redhat:enterprise_linux:8::baseos"},
					AddedBy:   "FakeLayer",
				},
				{
					Name:      "libsolv",
					Namespace: "rhel:8",
					Version:   "0.7.7-1.el8.x86_64",
					Arch:      "x86_64",
					Module:    "FakeMod",
					AddedBy:   "FakeLayer",
				},
			},
			LanguageComponents: nil,
		},
		Notes: nil,
	}
	return msg, nil
}
