package nodeinventorizer

import (
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// FakeNodeInventorizer can be used to send fake messages that would be emitted by NodeInventory
type FakeNodeInventorizer struct {
}

// Scan returns a fake message in the same format a real NodeInventory would produce
func (f *FakeNodeInventorizer) Scan(nodeName string) (*storage.NodeInventory, error) {
	log.Infof("Generating fake scan result message...")
	msg := &storage.NodeInventory{
		NodeName: nodeName,
		ScanTime: timestamp.TimestampNow(),
		Components: &storage.NodeInventory_Components{
			Namespace: "Testme OS",
			RhelComponents: []*storage.NodeInventory_Components_RHELComponent{
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
		},
		Notes: []storage.NodeInventory_Note{storage.NodeInventory_LANGUAGE_CVES_UNAVAILABLE},
	}
	return msg, nil
}
