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
					AddedBy:   "FakeLayer",
				},
				{
					Name:      "sqlite-libs",
					Namespace: "centos:8",
					Version:   "3.26.0-6.el8",
					AddedBy:   "FakeLayer",
				},
			},
			RhelComponents: []*scannerV1.RHELComponent{
				{
					Name:      "vim-minimal",
					Namespace: "rhel:7",
					Version:   "2:7.4.629-6.el7.x86_64",
					Arch:      "x86_64",
					Module:    "FakeMod",
					Cpes:      []string{"cpe:/a:redhat:enterprise_linux:8::baseos"},
					AddedBy:   "FakeLayer",
					Executables: []*scannerV1.Executable{
						{
							Path: "/usr/bin/vi",
							RequiredFeatures: []*scannerV1.FeatureNameVersion{
								{Name: "glibc", Version: "2.17-307.el7.1.i686"},
								{Name: "glibc", Version: "2.17-307.el7.1.x86_64"},
								{Name: "libacl", Version: "2.2.51-15.el7.x86_64"},
								{Name: "libattr", Version: "2.4.46-13.el7.i686"},
								{Name: "libattr", Version: "2.4.46-13.el7.x86_64"},
								{Name: "libselinux", Version: "2.5-15.el7.i686"},
								{Name: "libselinux", Version: "2.5-15.el7.x86_64"},
								{Name: "ncurses-libs", Version: "5.9-14.20130511.el7_4.x86_64"},
								{Name: "pcre", Version: "8.32-17.el7.i686"},
								{Name: "pcre", Version: "8.32-17.el7.x86_64"},
								{Name: "vim-minimal", Version: "2:7.4.629-6.el7.x86_64"},
							},
						},
					},
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
