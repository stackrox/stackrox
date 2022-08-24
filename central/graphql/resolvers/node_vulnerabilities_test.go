package resolvers

import (
	"context"
	"reflect"
	"testing"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v4/pgxpool"
	nodeCVEDataStore "github.com/stackrox/rox/central/cve/node/datastore"
	nodeCVESearch "github.com/stackrox/rox/central/cve/node/datastore/search"
	nodeCVEPostgres "github.com/stackrox/rox/central/cve/node/datastore/store/postgres"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	nodeDackboxDataStore "github.com/stackrox/rox/central/node/datastore/dackbox/datastore"
	nodeGlobalDataStore "github.com/stackrox/rox/central/node/datastore/dackbox/globaldatastore"
	nodeSearch "github.com/stackrox/rox/central/node/datastore/search"
	nodePostgres "github.com/stackrox/rox/central/node/datastore/store/postgres"
	nodeComponentDataStore "github.com/stackrox/rox/central/nodecomponent/datastore"
	nodeComponentSearch "github.com/stackrox/rox/central/nodecomponent/datastore/search"
	nodeComponentPostgres "github.com/stackrox/rox/central/nodecomponent/datastore/store/postgres"
	nodeComponentCVEEdgeDataStore "github.com/stackrox/rox/central/nodecomponentcveedge/datastore"
	nodeComponentCVEEdgeSearch "github.com/stackrox/rox/central/nodecomponentcveedge/datastore/search"
	nodeComponentCVEEdgePostgres "github.com/stackrox/rox/central/nodecomponentcveedge/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

func TestGraphQLNodeVulnerabilityEndpoints(t *testing.T) {
	suite.Run(t, new(GraphQLNodeVulnerabilityTestSuite))
}

type GraphQLNodeVulnerabilityTestSuite struct {
	suite.Suite

	ctx      context.Context
	db       *pgxpool.Pool
	gormDB   *gorm.DB
	resolver *Resolver

	envIsolator *envisolator.EnvIsolator
}

func (s *GraphQLNodeVulnerabilityTestSuite) SetupSuite() {
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
	nodePostgres.Destroy(s.ctx, s.db)
	nodeComponentPostgres.Destroy(s.ctx, s.db)
	nodeCVEPostgres.Destroy(s.ctx, s.db)
	nodeComponentCVEEdgePostgres.Destroy(s.ctx, s.db)

	// create mock resolvers, set relevant ones
	s.resolver = NewMock()

	// nodeCVE datastore
	nodeCVEStore := nodeCVEPostgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB)
	nodeCVEIndexer := nodeCVEPostgres.NewIndexer(s.db)
	nodeCVESearcher := nodeCVESearch.New(nodeCVEStore, nodeCVEIndexer)
	nodeCVEDatastore, err := nodeCVEDataStore.New(nodeCVEStore, nodeCVEIndexer, nodeCVESearcher, concurrency.NewKeyFence())
	s.NoError(err, "Failed to create nodeCVEDatastore")
	s.resolver.NodeCVEDataStore = nodeCVEDatastore

	// node datastore
	riskMock := mockRisks.NewMockDataStore(gomock.NewController(s.T()))
	nodeStore := nodePostgres.CreateTableAndNewStore(s.ctx, s.T(), s.db, s.gormDB, false)
	nodeIndexer := nodePostgres.NewIndexer(s.db)
	nodeSearcher := nodeSearch.NewV2(nodeStore, nodeIndexer)
	nodePostgresDataStore := nodeDackboxDataStore.NewWithPostgres(nodeStore, nodeIndexer, nodeSearcher, riskMock, ranking.NewRanker(), ranking.NewRanker())
	nodeGlobalDatastore, err := nodeGlobalDataStore.New(nodePostgresDataStore)
	s.NoError(err, "Failed to create nodeGlobalDatastore")
	s.resolver.NodeGlobalDataStore = nodeGlobalDatastore

	// nodeComponent datastore
	nodeCompStore := nodeComponentPostgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB)
	nodeCompIndexer := nodeComponentPostgres.NewIndexer(s.db)
	nodeCompSearcher := nodeComponentSearch.New(nodeCompStore, nodeCompIndexer)
	s.resolver.NodeComponentDataStore = nodeComponentDataStore.New(nodeCompStore, nodeCompIndexer, nodeCompSearcher, riskMock, ranking.NewRanker())

	// nodeComponentCVEEdge datastore
	nodeComponentCveEdgeStore := nodeComponentCVEEdgePostgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB)
	nodeCompontCveEdgeIndexer := nodeComponentCVEEdgePostgres.NewIndexer(s.db)
	nodeComponentCveEdgeSearcher := nodeComponentCVEEdgeSearch.New(nodeComponentCveEdgeStore, nodeCompontCveEdgeIndexer)
	nodeComponentCveEdgeDatastore, err := nodeComponentCVEEdgeDataStore.New(nodeComponentCveEdgeStore, nodeCompontCveEdgeIndexer, nodeComponentCveEdgeSearcher)
	s.NoError(err)
	s.resolver.NodeComponentCVEEdgeDataStore = nodeComponentCveEdgeDatastore

	// Sac permissions
	s.ctx = sac.WithAllAccess(s.ctx)

	// loaders used by graphql layer
	loaders.RegisterTypeFactory(reflect.TypeOf(storage.Node{}), func() interface{} {
		return loaders.NewNodeLoader(nodePostgresDataStore)
	})
	loaders.RegisterTypeFactory(reflect.TypeOf(storage.NodeComponent{}), func() interface{} {
		return loaders.NewNodeComponentLoader(s.resolver.NodeComponentDataStore)
	})
	loaders.RegisterTypeFactory(reflect.TypeOf(storage.NodeCVE{}), func() interface{} {
		return loaders.NewNodeCVELoader(s.resolver.NodeCVEDataStore)
	})
	s.ctx = loaders.WithLoaderContext(s.ctx)
}

func testNodes() []*storage.Node {
	t1, err := ptypes.TimestampProto(time.Unix(0, 1000))
	utils.CrashOnError(err)
	t2, err := ptypes.TimestampProto(time.Unix(0, 2000))
	utils.CrashOnError(err)
	return []*storage.Node{
		{
			Id:   "id1",
			Name: "name1",
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
			Id:   "id2",
			Name: "name2",
			SetCves: &storage.Node_Cves{
				Cves: 5,
			},
			Scan: &storage.NodeScan{
				ScanTime: t2,
			},
		},
	}
}
