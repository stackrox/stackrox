package resolvers

import (
	"context"
	"reflect"
	"testing"

	"github.com/graph-gophers/graphql-go"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	clusterPostgres "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	clusterHealthPostgres "github.com/stackrox/rox/central/cluster/store/clusterhealth/postgres"
	clusterCVEEdgeDataStore "github.com/stackrox/rox/central/clustercveedge/datastore"
	clusterCVEEdgePostgres "github.com/stackrox/rox/central/clustercveedge/datastore/store/postgres"
	clusterCVEEdgeSearch "github.com/stackrox/rox/central/clustercveedge/search"
	imageComponentCVEEdgeDS "github.com/stackrox/rox/central/componentcveedge/datastore"
	imageComponentCVEEdgePostgres "github.com/stackrox/rox/central/componentcveedge/datastore/store/postgres"
	imageComponentCVEEdgeSearch "github.com/stackrox/rox/central/componentcveedge/search"
	clusterCVEDataStore "github.com/stackrox/rox/central/cve/cluster/datastore"
	clusterCVESearch "github.com/stackrox/rox/central/cve/cluster/datastore/search"
	clusterCVEPostgres "github.com/stackrox/rox/central/cve/cluster/datastore/store/postgres"
	imageCVEDS "github.com/stackrox/rox/central/cve/image/datastore"
	imageCVESearch "github.com/stackrox/rox/central/cve/image/datastore/search"
	imageCVEPostgres "github.com/stackrox/rox/central/cve/image/datastore/store/postgres"
	nodeCVEDataStore "github.com/stackrox/rox/central/cve/node/datastore"
	nodeCVESearch "github.com/stackrox/rox/central/cve/node/datastore/search"
	nodeCVEPostgres "github.com/stackrox/rox/central/cve/node/datastore/store/postgres"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	imagePostgres "github.com/stackrox/rox/central/image/datastore/store/postgres"
	imageComponentDS "github.com/stackrox/rox/central/imagecomponent/datastore"
	imageComponentPostgres "github.com/stackrox/rox/central/imagecomponent/datastore/store/postgres"
	imageComponentSearch "github.com/stackrox/rox/central/imagecomponent/search"
	imageComponentEdgeDS "github.com/stackrox/rox/central/imagecomponentedge/datastore"
	imageCVEEdgeDS "github.com/stackrox/rox/central/imagecveedge/datastore"
	imageCVEEdgePostgres "github.com/stackrox/rox/central/imagecveedge/datastore/postgres"
	imageCVEEdgeSearch "github.com/stackrox/rox/central/imagecveedge/search"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	netEntitiesMocks "github.com/stackrox/rox/central/networkgraph/entity/datastore/mocks"
	netFlowsMocks "github.com/stackrox/rox/central/networkgraph/flow/datastore/mocks"
	nodeDS "github.com/stackrox/rox/central/node/datastore"
	nodeSearch "github.com/stackrox/rox/central/node/datastore/search"
	nodePostgres "github.com/stackrox/rox/central/node/datastore/store/postgres"
	nodeComponentDataStore "github.com/stackrox/rox/central/nodecomponent/datastore"
	nodeComponentSearch "github.com/stackrox/rox/central/nodecomponent/datastore/search"
	nodeComponentPostgres "github.com/stackrox/rox/central/nodecomponent/datastore/store/postgres"
	nodeComponentCVEEdgeDataStore "github.com/stackrox/rox/central/nodecomponentcveedge/datastore"
	nodeComponentCVEEdgeSearch "github.com/stackrox/rox/central/nodecomponentcveedge/datastore/search"
	nodeComponentCVEEdgePostgres "github.com/stackrox/rox/central/nodecomponentcveedge/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	k8srolebindingStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	connMgrMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	"github.com/stackrox/rox/central/views/imagecve"
	imagesView "github.com/stackrox/rox/central/views/images"
	"github.com/stackrox/rox/central/views/nodecve"
	"github.com/stackrox/rox/central/views/platformcve"
	"github.com/stackrox/rox/central/vulnmgmt/vulnerabilityrequest/cache"
	vulnReqDatastore "github.com/stackrox/rox/central/vulnmgmt/vulnerabilityrequest/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// SetupTestPostgresConn sets up postgres testDB for testing
func SetupTestPostgresConn(t testing.TB) *pgtest.TestPostgres {
	return pgtest.ForT(t)
}

// SetupTestResolver creates a graphQL resolver and schema for testing
func SetupTestResolver(t testing.TB, datastores ...interface{}) (*Resolver, *graphql.Schema) {
	resolver := &Resolver{}
	for _, datastoreI := range datastores {
		switch ds := datastoreI.(type) {
		case imageCVEDS.DataStore:
			registerImageCveLoader(t, ds)
			resolver.ImageCVEDataStore = ds
		case nodeCVEDataStore.DataStore:
			registerNodeCVELoader(t, ds)
			resolver.NodeCVEDataStore = ds
		case clusterCVEDataStore.DataStore:
			registerClusterCveLoader(t, ds)
			resolver.ClusterCVEDataStore = ds
		case imageComponentDS.DataStore:
			registerImageComponentLoader(t, ds)
			resolver.ImageComponentDataStore = ds
		case nodeComponentDataStore.DataStore:
			registerNodeComponentLoader(t, ds)
			resolver.NodeComponentDataStore = ds
		case imageDS.DataStore:
			var imageView imagesView.ImageView
			for _, di := range datastores {
				if view, ok := di.(imagesView.ImageView); ok {
					imageView = view
				}
			}
			registerImageLoader(t, ds, imageView)
			resolver.ImageDataStore = ds
		case deploymentDatastore.DataStore:
			resolver.DeploymentDataStore = ds
		case namespaceDataStore.DataStore:
			resolver.NamespaceDataStore = ds
		case nodeDS.DataStore:
			registerNodeLoader(t, ds)
			resolver.NodeDataStore = ds
		case clusterDataStore.DataStore:
			resolver.ClusterDataStore = ds
		case vulnReqDatastore.DataStore:
			resolver.vulnReqStore = ds
		case imageCVEEdgeDS.DataStore:
			resolver.ImageCVEEdgeDataStore = ds
		case clusterCVEEdgeDataStore.DataStore:
			resolver.ClusterCVEEdgeDataStore = ds
		case imageComponentCVEEdgeDS.DataStore:
			resolver.ComponentCVEEdgeDataStore = ds
		case imageComponentEdgeDS.DataStore:
			resolver.ImageComponentEdgeDataStore = ds
		case nodeComponentCVEEdgeDataStore.DataStore:
			resolver.NodeComponentCVEEdgeDataStore = ds
		case k8srolebindingStore.DataStore:
			resolver.K8sRoleBindingStore = ds

		case imagecve.CveView:
			resolver.ImageCVEView = ds
		case platformcve.CveView:
			resolver.PlatformCVEView = ds
		case nodecve.CveView:
			resolver.NodeCVEView = ds
		}
	}

	schema, err := graphql.ParseSchema(Schema(), resolver)
	assert.NoError(t, err)

	return resolver, schema
}

// CreateTestImageDatastore creates image datastore for testing
func CreateTestImageDatastore(t testing.TB, testDB *pgtest.TestPostgres, ctrl *gomock.Controller) imageDS.DataStore {
	ctx := context.Background()
	imagePostgres.Destroy(ctx, testDB.DB)

	risks := mockRisks.NewMockDataStore(ctrl)
	risks.EXPECT().RemoveRisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	return imageDS.NewWithPostgres(
		imagePostgres.CreateTableAndNewStore(ctx, testDB.DB, testDB.GetGormDB(t), false),
		risks,
		ranking.NewRanker(),
		ranking.NewRanker(),
	)
}

// CreateTestImageComponentDatastore creates imageComponent datastore for testing
func CreateTestImageComponentDatastore(t testing.TB, testDB *pgtest.TestPostgres, ctrl *gomock.Controller) imageComponentDS.DataStore {
	ctx := context.Background()
	imageComponentPostgres.Destroy(ctx, testDB.DB)

	mockRisk := mockRisks.NewMockDataStore(ctrl)
	storage := imageComponentPostgres.CreateTableAndNewStore(ctx, testDB.DB, testDB.GetGormDB(t))
	searcher := imageComponentSearch.NewV2(storage)

	return imageComponentDS.New(storage, searcher, mockRisk, ranking.NewRanker())
}

// CreateTestImageCVEDatastore creates imageCVE datastore for testing
func CreateTestImageCVEDatastore(t testing.TB, testDB *pgtest.TestPostgres) imageCVEDS.DataStore {
	ctx := context.Background()
	imageCVEPostgres.Destroy(ctx, testDB.DB)

	storage := imageCVEPostgres.CreateTableAndNewStore(ctx, testDB.DB, testDB.GetGormDB(t))
	searcher := imageCVESearch.New(storage)
	datastore := imageCVEDS.New(storage, searcher, nil)

	return datastore
}

// CreateTestImageComponentCVEEdgeDatastore creates edge datastore for edge table between imageComponent and imageCVE
func CreateTestImageComponentCVEEdgeDatastore(t testing.TB, testDB *pgtest.TestPostgres) imageComponentCVEEdgeDS.DataStore {
	ctx := context.Background()
	imageComponentCVEEdgePostgres.Destroy(ctx, testDB.DB)

	storage := imageComponentCVEEdgePostgres.CreateTableAndNewStore(ctx, testDB.DB, testDB.GetGormDB(t))
	searcher := imageComponentCVEEdgeSearch.NewV2(storage)

	return imageComponentCVEEdgeDS.New(storage, searcher)
}

// CreateTestImageComponentEdgeDatastore creates edge datastore for edge table between image and imageComponent
func CreateTestImageComponentEdgeDatastore(t testing.TB, testDB *pgtest.TestPostgres) imageComponentEdgeDS.DataStore {
	ds, err := imageComponentEdgeDS.GetTestPostgresDataStore(t, testDB.DB)
	assert.NoError(t, err)
	return ds
}

// CreateTestImageCVEEdgeDatastore creates edge datastore for edge table between image and imageCVE
func CreateTestImageCVEEdgeDatastore(t testing.TB, testDB *pgtest.TestPostgres) imageCVEEdgeDS.DataStore {
	ctx := context.Background()
	imageCVEEdgePostgres.Destroy(ctx, testDB.DB)

	storage := imageCVEEdgePostgres.CreateTableAndNewStore(ctx, testDB.DB, testDB.GetGormDB(t))
	searcher := imageCVEEdgeSearch.NewV2(storage)
	return imageCVEEdgeDS.New(storage, searcher)
}

// CreateTestDeploymentDatastore creates deployment datastore for testing
func CreateTestDeploymentDatastore(t testing.TB, testDB *pgtest.TestPostgres, ctrl *gomock.Controller, imageDatastore imageDS.DataStore) deploymentDatastore.DataStore {
	mockRisk := mockRisks.NewMockDataStore(ctrl)
	ds, err := deploymentDatastore.NewTestDataStore(
		t,
		testDB,
		&deploymentDatastore.DeploymentTestStoreParams{
			ImagesDataStore:  imageDatastore,
			RisksDataStore:   mockRisk,
			ClusterRanker:    ranking.ClusterRanker(),
			NamespaceRanker:  ranking.NamespaceRanker(),
			DeploymentRanker: ranking.DeploymentRanker(),
		},
	)
	assert.NoError(t, err)
	return ds
}

// CreateTestClusterCVEDatastore creates clusterCVE datastore for testing
func CreateTestClusterCVEDatastore(t testing.TB, testDB *pgtest.TestPostgres) clusterCVEDataStore.DataStore {
	ctx := context.Background()
	clusterCVEPostgres.Destroy(ctx, testDB.DB)

	storage := clusterCVEPostgres.NewFullTestStore(t, testDB.DB, clusterCVEPostgres.CreateTableAndNewStore(ctx, testDB.DB, testDB.GetGormDB(t)))
	searcher := clusterCVESearch.New(storage)
	datastore, err := clusterCVEDataStore.New(storage, searcher)
	assert.NoError(t, err, "failed to create cluster CVE datastore")
	return datastore
}

// CreateTestClusterCVEEdgeDatastore creates edge datastore for edge table between cluster and clusterCVE
func CreateTestClusterCVEEdgeDatastore(t testing.TB, testDB *pgtest.TestPostgres) clusterCVEEdgeDataStore.DataStore {
	ctx := context.Background()
	clusterCVEEdgePostgres.Destroy(ctx, testDB.DB)

	storage := clusterCVEEdgePostgres.CreateTableAndNewStore(ctx, testDB.DB, testDB.GetGormDB(t))
	searcher := clusterCVEEdgeSearch.NewV2(storage)
	datastore, err := clusterCVEEdgeDataStore.New(storage, searcher)
	assert.NoError(t, err, "failed to create cluster-CVE edge datastore")
	return datastore
}

// CreateTestNamespaceDatastore creates namespace datastore for testing
func CreateTestNamespaceDatastore(t testing.TB, testDB *pgtest.TestPostgres) namespaceDataStore.DataStore {
	datastore := namespaceDataStore.NewTestDataStore(t, testDB, nil, ranking.NamespaceRanker())
	return datastore
}

// CreateTestClusterDatastore creates cluster datastore for testing
func CreateTestClusterDatastore(t testing.TB, testDB *pgtest.TestPostgres, ctrl *gomock.Controller,
	clusterCVEDS clusterCVEDataStore.DataStore, namespaceDS namespaceDataStore.DataStore, nodeDataStore nodeDS.DataStore) clusterDataStore.DataStore {
	ctx := context.Background()
	clusterPostgres.Destroy(ctx, testDB.DB)

	storage := clusterPostgres.CreateTableAndNewStore(ctx, testDB.DB, testDB.GetGormDB(t))

	netEntities := netEntitiesMocks.NewMockEntityDataStore(ctrl)
	netFlows := netFlowsMocks.NewMockClusterDataStore(ctrl)
	connMgr := connMgrMocks.NewMockManager(ctrl)
	netEntities.EXPECT().RegisterCluster(gomock.Any(), gomock.Any()).AnyTimes()
	netFlows.EXPECT().CreateFlowStore(gomock.Any(), gomock.Any()).Return(netFlowsMocks.NewMockFlowDataStore(ctrl), nil).AnyTimes()
	connMgr.EXPECT().GetConnection(gomock.Any()).AnyTimes()
	datastore, err := clusterDataStore.New(storage, clusterHealthPostgres.CreateTableAndNewStore(ctx, testDB.DB, testDB.GetGormDB(t)),
		clusterCVEDS, nil, nil, namespaceDS, nil, nodeDataStore, nil, nil,
		netFlows, netEntities, nil, nil, nil, connMgr, nil, ranking.ClusterRanker(), nil, nil)
	assert.NoError(t, err, "failed to create cluster datastore")
	return datastore
}

// CreateTestNodeCVEDatastore creates nodeCVE datastore for testing
func CreateTestNodeCVEDatastore(t testing.TB, testDB *pgtest.TestPostgres) nodeCVEDataStore.DataStore {
	ctx := context.Background()
	nodeCVEPostgres.Destroy(ctx, testDB.DB)

	storage := nodeCVEPostgres.CreateTableAndNewStore(ctx, testDB.DB, testDB.GetGormDB(t))
	searcher := nodeCVESearch.New(storage)
	datastore, err := nodeCVEDataStore.New(storage, searcher, concurrency.NewKeyFence())
	assert.NoError(t, err, "failed to create node CVE datastore")
	return datastore
}

// CreateTestNodeComponentDatastore creates nodeComponent datastore for testing
func CreateTestNodeComponentDatastore(t testing.TB, testDB *pgtest.TestPostgres, ctrl *gomock.Controller) nodeComponentDataStore.DataStore {
	ctx := context.Background()
	nodeComponentPostgres.Destroy(ctx, testDB.DB)

	mockRisk := mockRisks.NewMockDataStore(ctrl)
	storage := nodeComponentPostgres.CreateTableAndNewStore(ctx, testDB.DB, testDB.GetGormDB(t))
	searcher := nodeComponentSearch.New(storage)
	return nodeComponentDataStore.New(storage, searcher, mockRisk, ranking.NewRanker())
}

// CreateTestNodeDatastore creates node datastore for testing
func CreateTestNodeDatastore(t testing.TB, testDB *pgtest.TestPostgres, ctrl *gomock.Controller) nodeDS.DataStore {
	ctx := context.Background()
	nodePostgres.Destroy(ctx, testDB.DB)

	mockRisk := mockRisks.NewMockDataStore(ctrl)
	storage := nodePostgres.CreateTableAndNewStore(ctx, t, testDB.DB, testDB.GetGormDB(t), false)
	searcher := nodeSearch.NewV2(storage)
	return nodeDS.NewWithPostgres(storage, searcher, mockRisk, ranking.NewRanker(), ranking.NewRanker())
}

// CreateTestNodeComponentCveEdgeDatastore creates edge datastore for edge table between nodeComponent and nodeCVE
func CreateTestNodeComponentCveEdgeDatastore(t testing.TB, testDB *pgtest.TestPostgres) nodeComponentCVEEdgeDataStore.DataStore {
	ctx := context.Background()
	nodeComponentCVEEdgePostgres.Destroy(ctx, testDB.DB)

	storage := nodeComponentCVEEdgePostgres.CreateTableAndNewStore(ctx, testDB.DB, testDB.GetGormDB(t))
	searcher := nodeComponentCVEEdgeSearch.New(storage)
	return nodeComponentCVEEdgeDataStore.New(storage, searcher)
}

// TestVulnReqDatastore return test vulnerability request datastore.
func TestVulnReqDatastore(t testing.TB, testDB *pgtest.TestPostgres) (vulnReqDatastore.DataStore, error) {
	return vulnReqDatastore.GetTestPostgresDataStore(t, testDB, cache.New(), cache.New())
}

func registerImageLoader(_ testing.TB, ds imageDS.DataStore, view imagesView.ImageView) {
	loaders.RegisterTypeFactory(reflect.TypeOf(storage.Image{}), func() interface{} {
		return loaders.NewImageLoader(ds, view)
	})
}

func registerImageComponentLoader(_ testing.TB, ds imageComponentDS.DataStore) {
	loaders.RegisterTypeFactory(reflect.TypeOf(storage.ImageComponent{}), func() interface{} {
		return loaders.NewComponentLoader(ds)
	})
}

func registerImageCveLoader(_ testing.TB, ds imageCVEDS.DataStore) {
	loaders.RegisterTypeFactory(reflect.TypeOf(storage.ImageCVE{}), func() interface{} {
		return loaders.NewImageCVELoader(ds)
	})
}

func registerClusterCveLoader(_ testing.TB, ds clusterCVEDataStore.DataStore) {
	loaders.RegisterTypeFactory(reflect.TypeOf(storage.ClusterCVE{}), func() interface{} {
		return loaders.NewClusterCVELoader(ds)
	})
}

func registerNodeLoader(_ testing.TB, ds nodeDS.DataStore) {
	loaders.RegisterTypeFactory(reflect.TypeOf(storage.Node{}), func() interface{} {
		return loaders.NewNodeLoader(ds)
	})
}

func registerNodeComponentLoader(_ testing.TB, ds nodeComponentDataStore.DataStore) {
	loaders.RegisterTypeFactory(reflect.TypeOf(storage.NodeComponent{}), func() interface{} {
		return loaders.NewNodeComponentLoader(ds)
	})
}

func registerNodeCVELoader(_ testing.TB, ds nodeCVEDataStore.DataStore) {
	loaders.RegisterTypeFactory(reflect.TypeOf(storage.NodeCVE{}), func() interface{} {
		return loaders.NewNodeCVELoader(ds)
	})
}
