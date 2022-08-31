package resolvers

import (
	"context"
	"testing"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/graph-gophers/graphql-go"
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
								Cvss:     4,
							},
							{
								Cve:      "cve-2019-2",
								Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
								Cvss:     3,
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

func testNodes() []*storage.Node {
	t1, err := ptypes.TimestampProto(time.Unix(0, 1000))
	utils.CrashOnError(err)
	t2, err := ptypes.TimestampProto(time.Unix(0, 2000))
	utils.CrashOnError(err)
	return []*storage.Node{
		{
			Id:   "nodeID1",
			Name: "node1",
			SetCves: &storage.Node_Cves{
				Cves: 3,
			},
			Scan: &storage.NodeScan{
				ScanTime: t1,
				Components: []*storage.EmbeddedNodeScanComponent{
					{
						Name:    "comp1",
						Version: "0.9",
						Vulnerabilities: []*storage.NodeVulnerability{
							{
								CveBaseInfo: &storage.CVEInfo{
									Cve: "cve-2018-1",
								},
								SetFixedBy: &storage.NodeVulnerability_FixedBy{
									FixedBy: "1.1",
								},
							},
						},
					},
					{
						Name:    "comp2",
						Version: "1.1",
						Vulnerabilities: []*storage.NodeVulnerability{
							{
								CveBaseInfo: &storage.CVEInfo{
									Cve: "cve-2018-1",
								},
								SetFixedBy: &storage.NodeVulnerability_FixedBy{
									FixedBy: "1.5",
								},
							},
						},
					},
					{
						Name:    "comp3",
						Version: "1.0",
						Vulnerabilities: []*storage.NodeVulnerability{
							{
								CveBaseInfo: &storage.CVEInfo{
									Cve: "cve-2019-1",
								},
								Cvss: 4,
							},
							{
								CveBaseInfo: &storage.CVEInfo{
									Cve: "cve-2019-2",
								},
								Cvss: 3,
							},
						},
					},
				},
			},
		},
		{
			Id:   "nodeID2",
			Name: "node2",
			SetCves: &storage.Node_Cves{
				Cves: 5,
			},
			Scan: &storage.NodeScan{
				ScanTime: t2,
				Components: []*storage.EmbeddedNodeScanComponent{
					{
						Name:    "comp1",
						Version: "0.9",
						Vulnerabilities: []*storage.NodeVulnerability{
							{
								CveBaseInfo: &storage.CVEInfo{
									Cve: "cve-2018-1",
								},
								SetFixedBy: &storage.NodeVulnerability_FixedBy{
									FixedBy: "1.1",
								},
								Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
							},
						},
					},
					{
						Name:    "comp3",
						Version: "1.0",
						Vulnerabilities: []*storage.NodeVulnerability{
							{
								CveBaseInfo: &storage.CVEInfo{
									Cve: "cve-2019-1",
								},
								Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
								Cvss:     4,
							},
							{
								CveBaseInfo: &storage.CVEInfo{
									Cve: "cve-2019-2",
								},
								Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
								Cvss:     3,
							},
						},
					},
					{
						Name:    "comp4",
						Version: "1.0",
						Vulnerabilities: []*storage.NodeVulnerability{
							{
								CveBaseInfo: &storage.CVEInfo{
									Cve: "cve-2017-1",
								},
								Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
							},
							{
								CveBaseInfo: &storage.CVEInfo{
									Cve: "cve-2017-2",
								},
								Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
							},
						},
					},
				},
			},
		},
	}
}

// returns clusters and associated nodes for testing
func testClustersWithNodes() ([]*storage.Cluster, []*storage.Node) {
	clusters := []*storage.Cluster{
		{
			Id:        "clusterID1",
			Name:      "cluster1",
			MainImage: "quay.io/stackrox-io/main",
		},
		{
			Id:        "clusterID2",
			Name:      "cluster2",
			MainImage: "quay.io/stackrox-io/main",
		},
	}

	nodes := testNodes()
	nodes[0].ClusterId = clusters[0].Id
	nodes[0].ClusterName = clusters[0].Name

	nodes[1].ClusterId = clusters[1].Id
	nodes[1].ClusterName = clusters[1].Name

	return clusters, nodes
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

func getIDList(ctx context.Context, resolvers interface{}) []string {
	var list []string
	switch res := resolvers.(type) {
	case []ImageVulnerabilityResolver:
		for _, r := range res {
			list = append(list, string(r.Id(ctx)))
		}
	case []*imageResolver:
		for _, r := range res {
			list = append(list, string(r.Id(ctx)))
		}
	case []ImageComponentResolver:
		for _, r := range res {
			list = append(list, string(r.Id(ctx)))
		}
	case []NodeVulnerabilityResolver:
		for _, r := range res {
			list = append(list, string(r.Id(ctx)))
		}
	case []*nodeResolver:
		for _, r := range res {
			list = append(list, string(r.Id(ctx)))
		}
	case []NodeComponentResolver:
		for _, r := range res {
			list = append(list, string(r.Id(ctx)))
		}
	case []*clusterResolver:
		for _, r := range res {
			list = append(list, string(r.Id(ctx)))
		}
	}
	return list
}

func getClusterResolver(ctx context.Context, t *testing.T, resolver *Resolver, id string) *clusterResolver {
	clusterID := graphql.ID(id)

	cluster, err := resolver.Cluster(ctx, struct{ graphql.ID }{clusterID})
	require.NoError(t, err)
	require.Equal(t, clusterID, cluster.Id(ctx))
	return cluster
}

func getNodeResolver(ctx context.Context, t *testing.T, resolver *Resolver, id string) *nodeResolver {
	nodeID := graphql.ID(id)

	node, err := resolver.Node(ctx, struct{ graphql.ID }{nodeID})
	require.NoError(t, err)
	require.Equal(t, nodeID, node.Id(ctx))
	return node
}

func getNodeComponentResolver(ctx context.Context, t *testing.T, resolver *Resolver, id string) NodeComponentResolver {
	compID := graphql.ID(id)

	comp, err := resolver.NodeComponent(ctx, IDQuery{ID: &compID})
	require.NoError(t, err)
	require.Equal(t, compID, comp.Id(ctx))
	return comp
}

func getNodeVulnerabilityResolver(ctx context.Context, t *testing.T, resolver *Resolver, id string) NodeVulnerabilityResolver {
	vulnID := graphql.ID(id)

	vuln, err := resolver.NodeVulnerability(ctx, IDQuery{ID: &vulnID})
	require.NoError(t, err)
	require.Equal(t, vulnID, vuln.Id(ctx))
	return vuln
}
