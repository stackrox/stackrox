//go:build sql_integration
// +build sql_integration

package resolvers

import (
	"context"
	"reflect"
	"testing"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v4/pgxpool"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	clusterPostgres "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	clusterHealthPostgres "github.com/stackrox/rox/central/cluster/store/clusterhealth/postgres"
	clusterCVEEdgeDataStore "github.com/stackrox/rox/central/clustercveedge/datastore"
	clusterCVEEdgePostgres "github.com/stackrox/rox/central/clustercveedge/datastore/store/postgres"
	clusterCVEEdgeSearch "github.com/stackrox/rox/central/clustercveedge/search"
	clusterCVEDataStore "github.com/stackrox/rox/central/cve/cluster/datastore"
	clusterCVESearch "github.com/stackrox/rox/central/cve/cluster/datastore/search"
	clusterCVEPostgres "github.com/stackrox/rox/central/cve/cluster/datastore/store/postgres"
	"github.com/stackrox/rox/central/cve/converter/v2"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	namespacePostgres "github.com/stackrox/rox/central/namespace/store/postgres"
	netEntitiesMocks "github.com/stackrox/rox/central/networkgraph/entity/datastore/mocks"
	netFlowsMocks "github.com/stackrox/rox/central/networkgraph/flow/datastore/mocks"
	nodeMocks "github.com/stackrox/rox/central/node/globaldatastore/mocks"
	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

func TestGraphQLClusterVulnerabilityEndpoints(t *testing.T) {
	suite.Run(t, new(GraphQLClusterVulnerabilityTestSuite))
}

/*
Remaining TODO tasks:
*/

type GraphQLClusterVulnerabilityTestSuite struct {
	suite.Suite

	ctx      context.Context
	db       *pgxpool.Pool
	gormDB   *gorm.DB
	resolver *Resolver

	envIsolator *envisolator.EnvIsolator
}

func (s *GraphQLClusterVulnerabilityTestSuite) SetupSuite() {

	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")

	if !features.PostgresDatastore.Enabled() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}

	s.ctx = context.Background()

	source := pgtest.GetConnectionString(s.T())
	config, err := pgxpool.ParseConfig(source)
	s.NoError(err)

	pool, err := pgxpool.ConnectConfig(s.ctx, config)
	s.NoError(err)
	s.gormDB = pgtest.OpenGormDB(s.T(), source)
	s.db = pool

	// destroy datastores if they exist
	clusterCVEPostgres.Destroy(s.ctx, s.db)
	clusterCVEEdgePostgres.Destroy(s.ctx, s.db)
	clusterPostgres.Destroy(s.ctx, s.db)

	// create mock resolvers, set relevant ones
	s.resolver = NewMock()

	// clusterCVE datastore
	clusterCVEStore := clusterCVEPostgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB)
	clusterCVEFullStore := clusterCVEPostgres.NewFullTestStore(s.T(), s.db, clusterCVEStore)
	clusterCVEIndexer := clusterCVEPostgres.NewIndexer(s.db)
	clusterCVESearcher := clusterCVESearch.New(clusterCVEFullStore, clusterCVEIndexer)
	clusterCVEDatastore, err := clusterCVEDataStore.New(clusterCVEFullStore, clusterCVEIndexer, clusterCVESearcher)
	s.NoError(err, "Failed to create ClusterCVEDatastore")
	s.resolver.ClusterCVEDataStore = clusterCVEDatastore

	// clusterCVEEdge datastore
	clusterCVEEdgeStore := clusterCVEEdgePostgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB)
	clusterCVEEdgeFullStore := clusterCVEEdgePostgres.NewFullTestStore(s.T(), clusterCVEEdgeStore)
	clusterCVEEdgeIndexer := clusterCVEEdgePostgres.NewIndexer(s.db)
	clusterCVEEdgeSearcher := clusterCVEEdgeSearch.NewV2(clusterCVEEdgeFullStore, clusterCVEEdgeIndexer)
	clusterCVEEdgeDatastore, err := clusterCVEEdgeDataStore.New(nil, clusterCVEEdgeFullStore, clusterCVEEdgeIndexer, clusterCVEEdgeSearcher)
	s.NoError(err, "Failed to create ClusterCVEEdgeDatastore")
	s.resolver.ClusterCVEEdgeDataStore = clusterCVEEdgeDatastore

	// namespace datastore
	namespaceStore := namespacePostgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB)
	namespaceIndexer := namespacePostgres.NewIndexer(s.db)
	namespaceDatastore, err := namespaceDataStore.New(namespaceStore, nil, namespaceIndexer, nil, ranking.NamespaceRanker(), nil)
	s.NoError(err, "Failed to create NamespaceDatastore")
	s.resolver.NamespaceDataStore = namespaceDatastore

	// cluster datastore
	clusterStore := clusterPostgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB)
	clusterIndexer := clusterPostgres.NewIndexer(s.db)

	mockCtrl := gomock.NewController(s.T())
	netEntities := netEntitiesMocks.NewMockEntityDataStore(mockCtrl)
	nodeDataStore := nodeMocks.NewMockGlobalDataStore(mockCtrl)
	netFlows := netFlowsMocks.NewMockClusterDataStore(mockCtrl)

	nodeDataStore.EXPECT().GetAllClusterNodeStores(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	netEntities.EXPECT().RegisterCluster(gomock.Any(), gomock.Any()).AnyTimes()
	netFlows.EXPECT().CreateFlowStore(gomock.Any(), gomock.Any()).Return(netFlowsMocks.NewMockFlowDataStore(mockCtrl), nil).AnyTimes()

	clusterDatastore, err := clusterDataStore.New(clusterStore, clusterHealthPostgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB), clusterCVEDatastore, nil, nil, namespaceDatastore, nil, nodeDataStore, nil, nil, netFlows, netEntities, nil, nil, nil, nil, nil, nil, ranking.ClusterRanker(), clusterIndexer, nil)
	s.NoError(err, "Failed to create ClusterDatastore")
	s.resolver.ClusterDataStore = clusterDatastore

	// Sac permissions
	s.ctx = sac.WithAllAccess(s.ctx)

	// loaders used by graphql layer
	loaders.RegisterTypeFactory(reflect.TypeOf(storage.ClusterCVE{}), func() interface{} {
		return loaders.NewClusterCVELoader(s.resolver.ClusterCVEDataStore)
	})
	s.ctx = loaders.WithLoaderContext(s.ctx)

	// Add Test Data to DataStores
	clusters := testCluster()
	clusterIDs := make([]string, 0, len(clusters))
	for _, c := range clusters {
		clusterID, err := clusterDatastore.AddCluster(s.ctx, c)
		s.NoError(err)
		clusterIDs = append(clusterIDs, clusterID)
	}

	clusterCVEParts := testClusterCVEParts(clusterIDs)
	for _, part := range clusterCVEParts {
		err := clusterCVEDatastore.UpsertClusterCVEsInternal(s.ctx, part.CVE.Type, part)
		s.NoError(err)
	}
}

func (s *GraphQLClusterVulnerabilityTestSuite) TearDownSuite() {
	s.envIsolator.RestoreAll()

	clusterCVEPostgres.Destroy(s.ctx, s.db)
	clusterCVEEdgePostgres.Destroy(s.ctx, s.db)
	clusterPostgres.Destroy(s.ctx, s.db)
	pgtest.CloseGormDB(s.T(), s.gormDB)
	s.db.Close()
}

func (s *GraphQLClusterVulnerabilityTestSuite) TestFoo() {

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
				Id:          cveIds[0],
				Cvss:        4,
				Type:        storage.CVE_K8S_CVE,
				CveBaseInfo: &storage.CVEInfo{CreatedAt: t1},
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
				Id:          cveIds[1],
				Cvss:        5,
				Type:        storage.CVE_K8S_CVE,
				CveBaseInfo: &storage.CVEInfo{CreatedAt: t1},
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
				Id:          cveIds[2],
				Cvss:        7,
				Type:        storage.CVE_K8S_CVE,
				CveBaseInfo: &storage.CVEInfo{CreatedAt: t2},
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
				Type:        storage.CVE_K8S_CVE,
				CveBaseInfo: &storage.CVEInfo{CreatedAt: t2},
			},
			Children: []converter.EdgeParts{
				{
					Edge: &storage.ClusterCVEEdge{
						Id:         postgres.IDFromPks([]string{clusterIDs[0], cveIds[3]}),
						IsFixable:  true,
						HasFixedBy: &storage.ClusterCVEEdge_FixedBy{FixedBy: "1.3"},
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
