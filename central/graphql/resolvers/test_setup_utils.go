package resolvers

import (
	"reflect"
	"testing"

	"github.com/graph-gophers/graphql-go"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	clusterPostgres "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	clusterHealthPostgres "github.com/stackrox/rox/central/cluster/store/clusterhealth/postgres"
	clusterCVEEdgeDataStore "github.com/stackrox/rox/central/clustercveedge/datastore"
	clusterCVEEdgePostgres "github.com/stackrox/rox/central/clustercveedge/datastore/store/postgres"
	clusterCVEDataStore "github.com/stackrox/rox/central/cve/cluster/datastore"
	clusterCVEPostgres "github.com/stackrox/rox/central/cve/cluster/datastore/store/postgres"
	imageCVEInfoDS "github.com/stackrox/rox/central/cve/image/info/datastore"
	imageCVEV2DS "github.com/stackrox/rox/central/cve/image/v2/datastore"
	imageCVEV2Postgres "github.com/stackrox/rox/central/cve/image/v2/datastore/store/postgres"
	nodeCVEDataStore "github.com/stackrox/rox/central/cve/node/datastore"
	nodeCVEPostgres "github.com/stackrox/rox/central/cve/node/datastore/store/postgres"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	imagePostgresV2 "github.com/stackrox/rox/central/image/datastore/store/v2/postgres"
	imageComponentV2DS "github.com/stackrox/rox/central/imagecomponent/v2/datastore"
	imageComponentV2Postgres "github.com/stackrox/rox/central/imagecomponent/v2/datastore/store/postgres"
	imageV2DS "github.com/stackrox/rox/central/imagev2/datastore"
	imageV2Postgres "github.com/stackrox/rox/central/imagev2/datastore/store/postgres"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	netEntitiesMocks "github.com/stackrox/rox/central/networkgraph/entity/datastore/mocks"
	netFlowsMocks "github.com/stackrox/rox/central/networkgraph/flow/datastore/mocks"
	nodeDS "github.com/stackrox/rox/central/node/datastore"
	nodePostgres "github.com/stackrox/rox/central/node/datastore/store/postgres"
	nodeComponentDataStore "github.com/stackrox/rox/central/nodecomponent/datastore"
	nodeComponentPostgres "github.com/stackrox/rox/central/nodecomponent/datastore/store/postgres"
	nodeComponentCVEEdgeDataStore "github.com/stackrox/rox/central/nodecomponentcveedge/datastore"
	nodeComponentCVEEdgePostgres "github.com/stackrox/rox/central/nodecomponentcveedge/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	k8srolebindingStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	connMgrMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	deploymentsView "github.com/stackrox/rox/central/views/deployments"
	"github.com/stackrox/rox/central/views/imagecomponentflat"
	"github.com/stackrox/rox/central/views/imagecve"
	"github.com/stackrox/rox/central/views/imagecveflat"
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
		case imageCVEV2DS.DataStore:
			registerImageCveV2Loader(t, ds)
			resolver.ImageCVEV2DataStore = ds
		case nodeCVEDataStore.DataStore:
			registerNodeCVELoader(t, ds)
			resolver.NodeCVEDataStore = ds
		case clusterCVEDataStore.DataStore:
			registerClusterCveLoader(t, ds)
			resolver.ClusterCVEDataStore = ds
		case imageComponentV2DS.DataStore:
			registerImageComponentV2Loader(t, ds)
			resolver.ImageComponentV2DataStore = ds
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
		case imageV2DS.DataStore:
			var imageView imagesView.ImageView
			for _, di := range datastores {
				if view, ok := di.(imagesView.ImageView); ok {
					imageView = view
				}
			}
			registerImageV2Loader(t, ds, imageView)
			resolver.ImageV2DataStore = ds
		case deploymentDatastore.DataStore:
			var deploymentView deploymentsView.DeploymentView
			for _, di := range datastores {
				if view, ok := di.(deploymentsView.DeploymentView); ok {
					deploymentView = view
				}
			}
			registerDeploymentLoader(t, ds, deploymentView)
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
		case clusterCVEEdgeDataStore.DataStore:
			resolver.ClusterCVEEdgeDataStore = ds
		case nodeComponentCVEEdgeDataStore.DataStore:
			resolver.NodeComponentCVEEdgeDataStore = ds
		case k8srolebindingStore.DataStore:
			resolver.K8sRoleBindingStore = ds

		case imagecve.CveView:
			resolver.ImageCVEView = ds
		case imagecveflat.CveFlatView:
			resolver.ImageCVEFlatView = ds
		case imagecomponentflat.ComponentFlatView:
			resolver.ImageComponentFlatView = ds
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

// CreateTestImageV2Datastore creates image datastore for testing
func CreateTestImageV2Datastore(t testing.TB, testDB *pgtest.TestPostgres, ctrl *gomock.Controller) imageDS.DataStore {
	risks := mockRisks.NewMockDataStore(ctrl)
	risks.EXPECT().RemoveRisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	imageCVEInfo := imageCVEInfoDS.GetTestPostgresDataStore(t, testDB.DB)
	return imageDS.NewWithPostgres(
		imagePostgresV2.New(testDB.DB, false, concurrency.NewKeyFence()),
		risks,
		ranking.NewRanker(),
		ranking.NewRanker(),
		imageCVEInfo,
	)
}

// CreateTestImageV2V2Datastore creates image datastore for testing
func CreateTestImageV2V2Datastore(_ testing.TB, testDB *pgtest.TestPostgres, ctrl *gomock.Controller) imageV2DS.DataStore {
	risks := mockRisks.NewMockDataStore(ctrl)
	risks.EXPECT().RemoveRisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	return imageV2DS.NewWithPostgres(
		imageV2Postgres.New(testDB.DB, false, concurrency.NewKeyFence()),
		risks,
		ranking.NewRanker(),
		ranking.NewRanker(),
	)
}

// CreateTestImageComponentV2Datastore creates imageComponent datastore for testing
func CreateTestImageComponentV2Datastore(_ testing.TB, testDB *pgtest.TestPostgres, ctrl *gomock.Controller) imageComponentV2DS.DataStore {
	mockRisk := mockRisks.NewMockDataStore(ctrl)
	storage := imageComponentV2Postgres.New(testDB.DB)

	return imageComponentV2DS.New(storage, mockRisk, ranking.NewRanker())
}

// CreateTestImageCVEV2Datastore creates imageCVE datastore for testing
func CreateTestImageCVEV2Datastore(_ testing.TB, testDB *pgtest.TestPostgres) imageCVEV2DS.DataStore {
	storage := imageCVEV2Postgres.New(testDB.DB)
	datastore := imageCVEV2DS.New(storage)

	return datastore
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

// CreateTestDeploymentDatastoreWithImageV2 creates deployment datastore for testing
func CreateTestDeploymentDatastoreWithImageV2(t testing.TB, testDB *pgtest.TestPostgres, ctrl *gomock.Controller, imageDatastore imageV2DS.DataStore) deploymentDatastore.DataStore {
	mockRisk := mockRisks.NewMockDataStore(ctrl)
	ds, err := deploymentDatastore.NewTestDataStore(
		t,
		testDB,
		&deploymentDatastore.DeploymentTestStoreParams{
			ImagesV2DataStore: imageDatastore,
			RisksDataStore:    mockRisk,
			ClusterRanker:     ranking.ClusterRanker(),
			NamespaceRanker:   ranking.NamespaceRanker(),
			DeploymentRanker:  ranking.DeploymentRanker(),
		},
	)
	assert.NoError(t, err)
	return ds
}

// CreateTestClusterCVEDatastore creates clusterCVE datastore for testing
func CreateTestClusterCVEDatastore(t testing.TB, testDB *pgtest.TestPostgres) clusterCVEDataStore.DataStore {
	baseStore := clusterCVEPostgres.New(testDB)
	storage := clusterCVEPostgres.NewFullTestStore(t, testDB.DB, baseStore)
	datastore, err := clusterCVEDataStore.New(storage)
	assert.NoError(t, err, "failed to create cluster CVE datastore")
	return datastore
}

// CreateTestClusterCVEEdgeDatastore creates edge datastore for edge table between cluster and clusterCVE
func CreateTestClusterCVEEdgeDatastore(t testing.TB, testDB *pgtest.TestPostgres) clusterCVEEdgeDataStore.DataStore {
	storage := clusterCVEEdgePostgres.New(testDB)
	datastore, err := clusterCVEEdgeDataStore.New(storage)
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

	storage := clusterPostgres.New(testDB)
	healthStorage := clusterHealthPostgres.New(testDB)

	netEntities := netEntitiesMocks.NewMockEntityDataStore(ctrl)
	netFlows := netFlowsMocks.NewMockClusterDataStore(ctrl)
	connMgr := connMgrMocks.NewMockManager(ctrl)
	netEntities.EXPECT().RegisterCluster(gomock.Any(), gomock.Any()).AnyTimes()
	netFlows.EXPECT().CreateFlowStore(gomock.Any(), gomock.Any()).Return(netFlowsMocks.NewMockFlowDataStore(ctrl), nil).AnyTimes()
	connMgr.EXPECT().GetConnection(gomock.Any()).AnyTimes()
	datastore, err := clusterDataStore.New(storage, healthStorage,
		clusterCVEDS, nil, nil, namespaceDS, nil, nodeDataStore, nil, nil,
		netFlows, netEntities, nil, nil, nil, connMgr, nil, ranking.ClusterRanker(), nil, nil)
	assert.NoError(t, err, "failed to create cluster datastore")
	return datastore
}

// CreateTestNodeCVEDatastore creates nodeCVE datastore for testing
func CreateTestNodeCVEDatastore(t testing.TB, testDB *pgtest.TestPostgres) nodeCVEDataStore.DataStore {
	storage := nodeCVEPostgres.New(testDB)
	datastore, err := nodeCVEDataStore.New(storage, concurrency.NewKeyFence())
	assert.NoError(t, err, "failed to create node CVE datastore")
	return datastore
}

// CreateTestNodeComponentDatastore creates nodeComponent datastore for testing
func CreateTestNodeComponentDatastore(_ testing.TB, testDB *pgtest.TestPostgres, ctrl *gomock.Controller) nodeComponentDataStore.DataStore {
	mockRisk := mockRisks.NewMockDataStore(ctrl)
	storage := nodeComponentPostgres.New(testDB)
	return nodeComponentDataStore.New(storage, mockRisk, ranking.NewRanker())
}

// CreateTestNodeDatastore creates node datastore for testing
func CreateTestNodeDatastore(_ testing.TB, testDB *pgtest.TestPostgres, ctrl *gomock.Controller) nodeDS.DataStore {
	mockRisk := mockRisks.NewMockDataStore(ctrl)
	storage := nodePostgres.New(testDB, false, concurrency.NewKeyFence())
	return nodeDS.NewWithPostgres(storage, mockRisk, ranking.NewRanker(), ranking.NewRanker())
}

// CreateTestNodeComponentCveEdgeDatastore creates edge datastore for edge table between nodeComponent and nodeCVE
func CreateTestNodeComponentCveEdgeDatastore(_ testing.TB, testDB *pgtest.TestPostgres) nodeComponentCVEEdgeDataStore.DataStore {
	storage := nodeComponentCVEEdgePostgres.New(testDB)
	return nodeComponentCVEEdgeDataStore.New(storage)
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

func registerImageV2Loader(_ testing.TB, ds imageV2DS.DataStore, view imagesView.ImageView) {
	loaders.RegisterTypeFactory(reflect.TypeOf(storage.ImageV2{}), func() interface{} {
		return loaders.NewImageV2Loader(ds, view)
	})
}

func registerImageComponentV2Loader(_ testing.TB, ds imageComponentV2DS.DataStore) {
	loaders.RegisterTypeFactory(reflect.TypeOf(storage.ImageComponentV2{}), func() interface{} {
		return loaders.NewComponentV2Loader(ds)
	})
}

func registerImageCveV2Loader(_ testing.TB, ds imageCVEV2DS.DataStore) {
	loaders.RegisterTypeFactory(reflect.TypeOf(storage.ImageCVEV2{}), func() interface{} {
		return loaders.NewImageCVEV2Loader(ds)
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

func registerDeploymentLoader(_ testing.TB, ds deploymentDatastore.DataStore, view deploymentsView.DeploymentView) {
	loaders.RegisterTypeFactory(reflect.TypeOf(storage.Deployment{}), func() interface{} {
		return loaders.NewDeploymentLoader(ds, view)
	})
}
