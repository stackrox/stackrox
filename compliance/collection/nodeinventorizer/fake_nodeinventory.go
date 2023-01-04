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
			Namespace: "Red Hat Enterprise Linux CoreOS 45.82.202008101249-0 (Ootpa)",
			RhelComponents: []*storage.NodeInventory_Components_RHELComponent{
				{
					Id:        int64(6661),
					Name:      "vim-minimal",
					Namespace: "rhel:8",
					Version:   "2:7.4.629-6.el8",
					Arch:      "x86_64",
					Module:    "",
					Cpes:      []string{"cpe:/a:redhat:enterprise_linux:8::baseos", "cpe:/o:redhat:enterprise_linux:8::coreos"},
					AddedBy:   "FakeNodeScanner",
				},
				{
					Id:        int64(6662),
					Name:      "tar",
					Namespace: "rhel:8",
					Version:   "1.27.1.el8",
					Arch:      "x86_64",
					Cpes: []string{
						"cpe:/a:redhat:enterprise_linux:8::appstream", "cpe:/a:redhat:rhel:8.3::appstream",
						"cpe:/a:redhat:enterprise_linux:8::baseos", "cpe:/a:redhat:rhel:8.3::baseos",
					},
					Module:  "",
					AddedBy: "FakeNodeScanner",
				},
				{
					Id:        int64(6663),
					Name:      "lz4-libs",
					Namespace: "rhel:8",
					Version:   "1.8.3-3.el8_4",
					Arch:      "x86_64",
					Module:    "NoModule",
					Cpes: []string{
						"cpe:/a:redhat:enterprise_linux:8::appstream", "cpe:/a:redhat:rhel:8.3::appstream",
						"cpe:/a:redhat:enterprise_linux:8::baseos", "cpe:/a:redhat:rhel:8.3::baseos",
					},
					AddedBy: "FakeNodeScanner",
				},
				{
					Id:        int64(6664),
					Name:      "libksba",
					Namespace: "rhel:8",
					Version:   "1.3.5-7.el8",
					Arch:      "x86_64",
					Module:    "",
					Cpes: []string{
						"cpe:/a:redhat:enterprise_linux:8::appstream", "cpe:/a:redhat:rhel:8.3::appstream",
						"cpe:/a:redhat:enterprise_linux:8::baseos", "cpe:/a:redhat:rhel:8.3::baseos",
					},
					AddedBy: "FakeNodeScanner",
				},
			},
		},
		Notes: []storage.NodeInventory_Note{storage.NodeInventory_LANGUAGE_CVES_UNAVAILABLE},
	}
	return msg, nil
}
