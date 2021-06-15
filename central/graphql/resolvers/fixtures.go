package resolvers

import (
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
)

func testImages() []*storage.Image {
	t1, err := ptypes.TimestampProto(time.Unix(0, 1000))
	utils.CrashOnError(err)
	t2, err := ptypes.TimestampProto(time.Unix(0, 2000))
	utils.CrashOnError(err)
	return []*storage.Image{
		{
			Id: "sha1",
			SetCves: &storage.Image_Cves{
				Cves: 3,
			},
			Scan: &storage.ImageScan{
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Name:    "comp1",
						Version: "0.9",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve: "cve-2018-1",
								SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
									FixedBy: "1.1",
								},
							},
						},
					},
					{
						Name:    "comp2",
						Version: "1.1",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve: "cve-2018-1",
								SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
									FixedBy: "1.5",
								},
							},
						},
					},
					{
						Name:    "comp3",
						Version: "1.0",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve: "cve-2019-1",
							},
							{
								Cve: "cve-2019-2",
							},
						},
					},
				},
				ScanTime: t1,
			},
		},
		{
			Id: "sha2",
			SetCves: &storage.Image_Cves{
				Cves: 5,
			},
			Scan: &storage.ImageScan{
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Name:    "comp1",
						Version: "0.9",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve: "cve-2018-1",
								SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
									FixedBy: "1.1",
								},
							},
						},
					},
					{
						Name:    "comp3",
						Version: "1.0",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve: "cve-2019-1",
							},
							{
								Cve: "cve-2019-2",
							},
						},
					},
					{
						Name:    "comp4",
						Version: "1.0",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve: "cve-2017-1",
							},
							{
								Cve: "cve-2017-2",
							},
						},
					},
				},
				ScanTime: t2,
			},
		},
	}
}
