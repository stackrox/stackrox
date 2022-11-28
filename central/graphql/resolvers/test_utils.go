package resolvers

import (
	"context"
	"testing"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/graph-gophers/graphql-go"
	"github.com/jackc/pgx/v4/pgxpool"
	componentCVEEdgeDataStore "github.com/stackrox/rox/central/componentcveedge/datastore"
	componentCVEEdgePostgres "github.com/stackrox/rox/central/componentcveedge/datastore/store/postgres"
	componentCVEEdgeSearch "github.com/stackrox/rox/central/componentcveedge/search"
	"github.com/stackrox/rox/central/cve/converter/v2"
	imageCVEDataStore "github.com/stackrox/rox/central/cve/image/datastore"
	imageCVESearch "github.com/stackrox/rox/central/cve/image/datastore/search"
	imageCVEPostgres "github.com/stackrox/rox/central/cve/image/datastore/store/postgres"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	deploymentPostgres "github.com/stackrox/rox/central/deployment/store/postgres"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	imagePostgres "github.com/stackrox/rox/central/image/datastore/store/postgres"
	imageComponentDataStore "github.com/stackrox/rox/central/imagecomponent/datastore"
	imageComponentPostgres "github.com/stackrox/rox/central/imagecomponent/datastore/store/postgres"
	imageComponentSearch "github.com/stackrox/rox/central/imagecomponent/search"
	imageCVEEdgeDataStore "github.com/stackrox/rox/central/imagecveedge/datastore"
	imageCVEEdgePostgres "github.com/stackrox/rox/central/imagecveedge/datastore/postgres"
	imageCVEEdgeSearch "github.com/stackrox/rox/central/imagecveedge/search"
	"github.com/stackrox/rox/central/ranking"
	riskDataStore "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/generated/storage"
	dackboxConcurrency "github.com/stackrox/rox/pkg/dackbox/concurrency"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	types2 "github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func testDeployments() []*storage.Deployment {
	return []*storage.Deployment{
		{
			Id:          fixtureconsts.Deployment1,
			Name:        "dep1name",
			Namespace:   "namespace1name",
			NamespaceId: "namespace1id",
			ClusterId:   "cluster1id",
			ClusterName: "cluster1name",
			Containers: []*storage.Container{
				{
					Name:  "container1name",
					Image: types2.ToContainerImage(testImages()[0]),
				},
				{
					Name:  "container2name",
					Image: types2.ToContainerImage(testImages()[1]),
				},
			},
		},
		{
			Id:          fixtureconsts.Deployment2,
			Name:        "dep2name",
			Namespace:   "namespace1name",
			NamespaceId: "namespace1id",
			ClusterId:   "cluster1id",
			ClusterName: "cluster1name",
			Containers: []*storage.Container{
				{
					Name:  "container1name",
					Image: types2.ToContainerImage(testImages()[0]),
				},
			},
		},
		{
			Id:          fixtureconsts.Deployment3,
			Name:        "dep3name",
			Namespace:   "namespace2name",
			NamespaceId: "namespace2id",
			ClusterId:   "cluster2id",
			ClusterName: "cluster2name",
			Containers: []*storage.Container{
				{
					Name:  "container1name",
					Image: types2.ToContainerImage(testImages()[1]),
				},
			},
		},
	}
}

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

func testCluster() []*storage.Cluster {
	mainImage := "docker.io/stackrox/rox:latest"
	centralEndpoint := "central.stackrox:443"
	return []*storage.Cluster{
		{
			Name:               "k8s_cluster1",
			Type:               storage.ClusterType_KUBERNETES_CLUSTER,
			MainImage:          mainImage,
			CentralApiEndpoint: centralEndpoint,
		},
		{
			Name:               "k8s_cluster2",
			Type:               storage.ClusterType_KUBERNETES_CLUSTER,
			MainImage:          mainImage,
			CentralApiEndpoint: centralEndpoint,
		},
		{
			Name:               "os_cluster1",
			Type:               storage.ClusterType_OPENSHIFT_CLUSTER,
			MainImage:          mainImage,
			CentralApiEndpoint: centralEndpoint,
		},
		{
			Name:               "os_cluster2",
			Type:               storage.ClusterType_OPENSHIFT_CLUSTER,
			MainImage:          mainImage,
			CentralApiEndpoint: centralEndpoint,
		},
		{
			Name:               "os4_cluster1",
			Type:               storage.ClusterType_OPENSHIFT4_CLUSTER,
			MainImage:          mainImage,
			CentralApiEndpoint: centralEndpoint,
		},
		{
			Name:               "os4_cluster2",
			Type:               storage.ClusterType_OPENSHIFT4_CLUSTER,
			MainImage:          mainImage,
			CentralApiEndpoint: centralEndpoint,
		},
		{
			Name:               "gen_cluster1",
			Type:               storage.ClusterType_GENERIC_CLUSTER,
			MainImage:          mainImage,
			CentralApiEndpoint: centralEndpoint,
		},
		{
			Name:               "gen_cluster2",
			Type:               storage.ClusterType_GENERIC_CLUSTER,
			MainImage:          mainImage,
			CentralApiEndpoint: centralEndpoint,
		},
	}
}

func testClusterCVEParts(clusterIDs []string) []converter.ClusterCVEParts {
	cveIds := []string{"clusterCve1", "clusterCve2", "clusterCve3", "clusterCve4", "clusterCve5"}
	t1, err := ptypes.TimestampProto(time.Unix(0, 1000))
	utils.CrashOnError(err)
	t2, err := ptypes.TimestampProto(time.Unix(0, 2000))
	utils.CrashOnError(err)
	return []converter.ClusterCVEParts{
		{
			CVE: &storage.ClusterCVE{
				Id:       cveIds[0],
				Cvss:     4,
				Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
				Type:     storage.CVE_K8S_CVE,
				CveBaseInfo: &storage.CVEInfo{
					CreatedAt: t1,
					CvssV2:    &storage.CVSSV2{},
				},
			},
			Children: []converter.EdgeParts{
				{
					Edge: &storage.ClusterCVEEdge{
						Id:         postgres.IDFromPks([]string{clusterIDs[0], cveIds[0]}),
						IsFixable:  true,
						HasFixedBy: &storage.ClusterCVEEdge_FixedBy{FixedBy: "1.1"},
						ClusterId:  clusterIDs[0],
						CveId:      cveIds[0],
					},
					ClusterID: clusterIDs[0],
				},
			},
		},
		{
			CVE: &storage.ClusterCVE{
				Id:       cveIds[1],
				Cvss:     5,
				Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
				Type:     storage.CVE_K8S_CVE,
				CveBaseInfo: &storage.CVEInfo{
					CreatedAt: t1,
					CvssV3:    &storage.CVSSV3{},
				},
			},
			Children: []converter.EdgeParts{
				{
					Edge: &storage.ClusterCVEEdge{
						Id:         postgres.IDFromPks([]string{clusterIDs[0], cveIds[1]}),
						IsFixable:  false,
						HasFixedBy: nil,
						ClusterId:  clusterIDs[0],
						CveId:      cveIds[1],
					},
					ClusterID: clusterIDs[0],
				},
				{
					Edge: &storage.ClusterCVEEdge{
						Id:         postgres.IDFromPks([]string{clusterIDs[1], cveIds[1]}),
						IsFixable:  false,
						HasFixedBy: nil,
						ClusterId:  clusterIDs[1],
						CveId:      cveIds[1],
					},
					ClusterID: clusterIDs[1],
				},
			},
		},
		{
			CVE: &storage.ClusterCVE{
				Id:       cveIds[2],
				Cvss:     7,
				Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
				Type:     storage.CVE_K8S_CVE,
				CveBaseInfo: &storage.CVEInfo{
					CreatedAt: t2,
					CvssV2:    &storage.CVSSV2{},
					CvssV3:    &storage.CVSSV3{},
				},
			},
			Children: []converter.EdgeParts{
				{
					Edge: &storage.ClusterCVEEdge{
						Id:         postgres.IDFromPks([]string{clusterIDs[1], cveIds[2]}),
						IsFixable:  true,
						HasFixedBy: &storage.ClusterCVEEdge_FixedBy{FixedBy: "1.2"},
						ClusterId:  clusterIDs[1],
						CveId:      cveIds[2],
					},
					ClusterID: clusterIDs[1],
				},
			},
		},
		{
			CVE: &storage.ClusterCVE{
				Id:          cveIds[3],
				Cvss:        2,
				Severity:    storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
				Type:        storage.CVE_K8S_CVE,
				CveBaseInfo: &storage.CVEInfo{CreatedAt: t2},
			},
			Children: []converter.EdgeParts{
				{
					Edge: &storage.ClusterCVEEdge{
						Id:         postgres.IDFromPks([]string{clusterIDs[0], cveIds[3]}),
						IsFixable:  false,
						HasFixedBy: nil,
						ClusterId:  clusterIDs[0],
						CveId:      cveIds[3],
					},
					ClusterID: clusterIDs[0],
				},
				{
					Edge: &storage.ClusterCVEEdge{
						Id:         postgres.IDFromPks([]string{clusterIDs[1], cveIds[3]}),
						IsFixable:  true,
						HasFixedBy: &storage.ClusterCVEEdge_FixedBy{FixedBy: "1.4"},
						ClusterId:  clusterIDs[1],
						CveId:      cveIds[3],
					},
					ClusterID: clusterIDs[1],
				},
			},
		},
		{
			CVE: &storage.ClusterCVE{
				Id:          cveIds[4],
				Cvss:        2,
				Severity:    storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
				Type:        storage.CVE_K8S_CVE,
				CveBaseInfo: &storage.CVEInfo{CreatedAt: t1},
			},
			Children: []converter.EdgeParts{
				{
					Edge: &storage.ClusterCVEEdge{
						Id:         postgres.IDFromPks([]string{clusterIDs[0], cveIds[4]}),
						IsFixable:  false,
						HasFixedBy: nil,
						ClusterId:  clusterIDs[0],
						CveId:      cveIds[4],
					},
					ClusterID: clusterIDs[0],
				},
			},
		},
	}
}

func testImagesWithOperatingSystems() []*storage.Image {
	ret := testImages()
	ret[0].Scan.OperatingSystem = "os1"
	ret[1].Scan.OperatingSystem = "os2"
	return ret
}

func testNodes() []*storage.Node {
	t1, err := ptypes.TimestampProto(time.Unix(0, 1000))
	utils.CrashOnError(err)
	t2, err := ptypes.TimestampProto(time.Unix(0, 2000))
	utils.CrashOnError(err)
	return []*storage.Node{
		{
			Id:   fixtureconsts.Node1,
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
			Id:   fixtureconsts.Node2,
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
			Id:        fixtureconsts.Cluster1,
			Name:      "cluster1",
			MainImage: "quay.io/stackrox-io/main",
		},
		{
			Id:        fixtureconsts.Cluster2,
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
	case []ClusterVulnerabilityResolver:
		for _, r := range res {
			list = append(list, string(r.Id(ctx)))
		}
	case []*clusterResolver:
		for _, r := range res {
			list = append(list, string(r.Id(ctx)))
		}
	case []*deploymentResolver:
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

func getImageCVEDatastore(ctx context.Context, db *pgxpool.Pool, gormDB *gorm.DB) (imageCVEDataStore.DataStore, error) {
	imageCVEStore := imageCVEPostgres.CreateTableAndNewStore(ctx, db, gormDB)
	imageCVEIndexer := imageCVEPostgres.NewIndexer(db)
	imageCVESearcher := imageCVESearch.New(imageCVEStore, imageCVEIndexer)
	imageCVEDatastore, err := imageCVEDataStore.New(imageCVEStore, imageCVEIndexer, imageCVESearcher, dackboxConcurrency.NewKeyFence())
	return imageCVEDatastore, err
}

func getImageDatastore(ctx context.Context, db *pgxpool.Pool, gormDB *gorm.DB, risks riskDataStore.DataStore) imageDatastore.DataStore {
	imageStore := imagePostgres.CreateTableAndNewStore(ctx, db, gormDB, false)
	return imageDatastore.NewWithPostgres(imageStore, imagePostgres.NewIndexer(db), risks, ranking.NewRanker(), ranking.NewRanker())
}

func getImageComponentDatastore(ctx context.Context, db *pgxpool.Pool, gormDB *gorm.DB, risks riskDataStore.DataStore) imageComponentDataStore.DataStore {
	imageCompStore := imageComponentPostgres.CreateTableAndNewStore(ctx, db, gormDB)
	imageCompIndexer := imageComponentPostgres.NewIndexer(db)
	imageCompSearcher := imageComponentSearch.NewV2(imageCompStore, imageCompIndexer)
	return imageComponentDataStore.New(nil, imageCompStore, imageCompIndexer, imageCompSearcher, risks, ranking.NewRanker())
}
func getImageCVEEdgeDatastore(ctx context.Context, db *pgxpool.Pool, gormDB *gorm.DB) imageCVEEdgeDataStore.DataStore {
	imageCveEdgeStore := imageCVEEdgePostgres.CreateTableAndNewStore(ctx, db, gormDB)
	imageCveEdgeIndexer := imageCVEEdgePostgres.NewIndexer(db)
	imageCveEdgeSearcher := imageCVEEdgeSearch.NewV2(imageCveEdgeStore, imageCveEdgeIndexer)
	return imageCVEEdgeDataStore.New(nil, imageCveEdgeStore, imageCveEdgeSearcher)
}

func getImageComponentCVEEdgeDatastore(ctx context.Context, db *pgxpool.Pool, gormDB *gorm.DB) componentCVEEdgeDataStore.DataStore {
	componentCveEdgeStore := componentCVEEdgePostgres.CreateTableAndNewStore(ctx, db, gormDB)
	componentCveEdgeIndexer := componentCVEEdgePostgres.NewIndexer(db)
	componentCveEdgeSearcher := componentCVEEdgeSearch.NewV2(componentCveEdgeStore, componentCveEdgeIndexer)
	return componentCVEEdgeDataStore.New(nil, componentCveEdgeStore, componentCveEdgeIndexer, componentCveEdgeSearcher)
}

func getDeploymentDatastore(ctx context.Context, t *testing.T, db *pgxpool.Pool, gormDB *gorm.DB, imageDatastore imageDatastore.DataStore, risks riskDataStore.DataStore) (deploymentDatastore.DataStore, error) {
	deploymentStore := deploymentPostgres.NewFullTestStore(t, deploymentPostgres.CreateTableAndNewStore(ctx, db, gormDB))
	return deploymentDatastore.NewTestDataStore(t, deploymentStore, nil, db, nil, nil, imageDatastore, nil, nil, risks, nil, nil, ranking.ClusterRanker(), ranking.NamespaceRanker(), ranking.DeploymentRanker())
}
