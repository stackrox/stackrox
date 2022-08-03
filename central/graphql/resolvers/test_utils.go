package resolvers

import (
	"context"
	"testing"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/require"
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
								Cve:  "cve-2019-1",
								Cvss: 4,
							},
							{
								Cve:  "cve-2019-2",
								Cvss: 3,
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
								Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
							},
						},
					},
					{
						Name:    "comp3",
						Version: "1.0",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve:      "cve-2019-1",
								Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
							},
							{
								Cve:      "cve-2019-2",
								Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
							},
						},
					},
					{
						Name:    "comp4",
						Version: "1.0",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve:      "cve-2017-1",
								Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
							},
							{
								Cve:      "cve-2017-2",
								Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
							},
						},
					},
				},
				ScanTime: t2,
			},
		},
	}
}

func checkVulnerabilityCounter(t *testing.T, resolver *VulnerabilityCounterResolver, total, fixable, critical, important, moderate, low int32) {
	// we have to pass a context to the resolver functions because style checks don't like when we pass nil, this value isn't used though
	ctx := context.Background()
	require.Equal(t, total, resolver.All(ctx).Total(ctx))
	require.Equal(t, fixable, resolver.All(ctx).Fixable(ctx))
	require.Equal(t, critical, resolver.Critical(ctx).Total(ctx))
	require.Equal(t, important, resolver.Important(ctx).Total(ctx))
	require.Equal(t, moderate, resolver.Moderate(ctx).Total(ctx))
	require.Equal(t, low, resolver.Low(ctx).Total(ctx))
}

func getFixableRawQuery(fixable bool) (string, error) {
	return search.NewQueryBuilder().AddBools(search.Fixable, fixable).RawQuery()
}
