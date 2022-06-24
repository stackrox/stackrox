package testutils

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/jackc/pgx/v4/pgxpool"
	activeComponentDackbox "github.com/stackrox/rox/central/activecomponent/dackbox"
	activeComponentIndex "github.com/stackrox/rox/central/activecomponent/index"
	clusterCVEEdgeDackbox "github.com/stackrox/rox/central/clustercveedge/dackbox"
	clusterCVEEdgeIndex "github.com/stackrox/rox/central/clustercveedge/index"
	componentCVEEdgeDackbox "github.com/stackrox/rox/central/componentcveedge/dackbox"
	componentCVEEdgeIndex "github.com/stackrox/rox/central/componentcveedge/index"
	cveDackbox "github.com/stackrox/rox/central/cve/dackbox"
	cveIndex "github.com/stackrox/rox/central/cve/index"
	deploymentDackbox "github.com/stackrox/rox/central/deployment/dackbox"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	deploymentIndex "github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/globalindex"
	imageDackbox "github.com/stackrox/rox/central/image/dackbox"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	imageIndex "github.com/stackrox/rox/central/image/index"
	imageComponentDackbox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	imageComponentIndex "github.com/stackrox/rox/central/imagecomponent/index"
	imageComponentEdgeDackbox "github.com/stackrox/rox/central/imagecomponentedge/dackbox"
	imageComponentEdgeIndex "github.com/stackrox/rox/central/imagecomponentedge/index"
	imageCVEEdgeDackbox "github.com/stackrox/rox/central/imagecveedge/dackbox"
	imageCVEEdgeIndex "github.com/stackrox/rox/central/imagecveedge/index"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	nodeDackbox "github.com/stackrox/rox/central/node/dackbox"
	nodeDataStore "github.com/stackrox/rox/central/node/datastore/dackbox/datastore"
	nodeIndex "github.com/stackrox/rox/central/node/index"
	nodeComponentEdgeDackbox "github.com/stackrox/rox/central/nodecomponentedge/dackbox"
	nodeComponentEdgeIndex "github.com/stackrox/rox/central/nodecomponentedge/index"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/indexer"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	rocksPkg "github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/uuid"
)

type DackboxTestDataStore interface {
	// Expose internal for the case other datastores would be needed for testing purposes
	GetPostgresPool() *pgxpool.Pool
	GetRocksEngine() *rocksPkg.RocksDB
	GetBleveIndex() bleve.Index
	GetDackbox() *dackbox.DackBox
	GetKeyFence() concurrency.KeyFence
	// Data injection
	PushImageToVulnerabilitiesGraph() error
	PushNodeToVulnerabilitiesGraph() error
	// Post test cleanup (TearDown)
	Cleanup() error
}

type dackboxTestDataStoreImpl struct {
	// Pool for postgres mode
	pool *pgxpool.Pool
	// Elements for rocksdb+bleve mode
	rocksEngine *rocksPkg.RocksDB
	bleveIndex  bleve.Index
	dacky       *dackbox.DackBox
	keyFence    concurrency.KeyFence

	// DataStores
	namespaceStore  namespaceDataStore.DataStore
	deploymentStore deploymentDataStore.DataStore
	imageStore      imageDataStore.DataStore
	nodeStore       nodeDataStore.DataStore
}

func (s *dackboxTestDataStoreImpl) GetPostgresPool() *pgxpool.Pool {
	return s.pool
}

func (s *dackboxTestDataStoreImpl) GetRocksEngine() *rocksPkg.RocksDB {
	return s.rocksEngine
}

func (s *dackboxTestDataStoreImpl) GetBleveIndex() bleve.Index {
	return s.bleveIndex
}

func (s *dackboxTestDataStoreImpl) GetDackbox() *dackbox.DackBox {
	return s.dacky
}

func (s *dackboxTestDataStoreImpl) GetKeyFence() concurrency.KeyFence {
	return s.keyFence
}

func (s *dackboxTestDataStoreImpl) PushImageToVulnerabilitiesGraph() error {
	var err error
	ctx := sac.WithAllAccess(context.Background())
	testNamespace1 := fixtures.GetNamespace(testconsts.Cluster1, testconsts.Cluster1, testconsts.NamespaceA)
	testNamespace2 := fixtures.GetNamespace(testconsts.Cluster2, testconsts.Cluster2, testconsts.NamespaceB)
	testImage1 := fixtures.GetImageSherlockHolmes_1()
	testImage2 := fixtures.GetImageDoctorJekyll_2()
	testDeployment1 := fixtures.GetDeploymentSherlockHolmes_1(uuid.NewV4().String(), testNamespace1)
	testDeployment2 := fixtures.GetDeploymentDoctorJekyll_2(uuid.NewV4().String(), testNamespace2)
	err = s.namespaceStore.AddNamespace(ctx, testNamespace1)
	if err != nil {
		return err
	}
	err = s.namespaceStore.AddNamespace(ctx, testNamespace2)
	if err != nil {
		return err
	}
	err = s.imageStore.UpsertImage(ctx, testImage1)
	if err != nil {
		return err
	}
	err = s.imageStore.UpsertImage(ctx, testImage2)
	if err != nil {
		return err
	}
	err = s.deploymentStore.UpsertDeployment(ctx, testDeployment1)
	if err != nil {
		return err
	}
	err = s.deploymentStore.UpsertDeployment(ctx, testDeployment2)
	if err != nil {
		return err
	}
	return nil
}

func (s *dackboxTestDataStoreImpl) PushNodeToVulnerabilitiesGraph() error {
	var err error
	ctx := sac.WithAllAccess(context.Background())
	testNode1 := fixtures.GetScopedNode_1(uuid.NewV4().String(), testconsts.Cluster1)
	testNode2 := fixtures.GetScopedNode_2(uuid.NewV4().String(), testconsts.Cluster2)
	err = s.nodeStore.UpsertNode(ctx, testNode1)
	if err != nil {
		return err
	}
	err = s.nodeStore.UpsertNode(ctx, testNode2)
	if err != nil {
		return err
	}
	return nil

}

func (s *dackboxTestDataStoreImpl) Cleanup() error {
	if features.PostgresDatastore.Enabled() {
		s.pool.Close()
		return nil
	} else {
		var err error
		err = s.bleveIndex.Close()
		if err != nil {
			return err
		}
		err = rocksPkg.CloseAndRemove(s.rocksEngine)
		if err != nil {
			return err
		}
		return nil
	}
}

func NewDackboxTestDataStore(t *testing.T) (DackboxTestDataStore, error) {
	var err error
	s := &dackboxTestDataStoreImpl{}
	if features.PostgresDatastore.Enabled() {
		ctx := context.Background()
		configSrc := pgtest.GetConnectionString(t)
		s.pool, err = pgtest.GetPostgresPool(ctx, t)
		if err != nil {
			return nil, err
		}
		gormDB := pgtest.OpenGormDB(t, configSrc)
		defer pgtest.CloseGormDB(t, gormDB)
		s.nodeStore = nodeDataStore.GetTestPostgresDataStore(ctx, t, s.pool, gormDB)
		s.imageStore = imageDataStore.GetTestPostgresDataStore(ctx, t, s.pool, gormDB)
		s.deploymentStore = deploymentDataStore.GetTestPostgresDataStore(ctx, t, s.pool, gormDB)
		s.namespaceStore = namespaceDataStore.GetTestPostgresDataStore(ctx, t, s.pool, gormDB)
	} else {
		s.rocksEngine, err = rocksPkg.NewTemp("dackboxtest")
		if err != nil {
			return nil, err
		}
		s.bleveIndex, err = globalindex.MemOnlyIndex()
		if err != nil {
			return nil, err
		}
		s.keyFence = concurrency.NewKeyFence()
		indexQ := queue.NewWaitableQueue()
		s.dacky, err = dackbox.NewRocksDBDackBox(s.rocksEngine, indexQ, []byte("graph"), []byte("dirty"), []byte("valid"))
		if err != nil {
			return nil, err
		}
		reg := indexer.NewWrapperRegistry()
		indexer.NewLazy(indexQ, reg, s.bleveIndex, s.dacky.AckIndexed).Start()
		reg.RegisterWrapper(activeComponentDackbox.Bucket, activeComponentIndex.Wrapper{})
		reg.RegisterWrapper(clusterCVEEdgeDackbox.Bucket, clusterCVEEdgeIndex.Wrapper{})
		reg.RegisterWrapper(componentCVEEdgeDackbox.Bucket, componentCVEEdgeIndex.Wrapper{})
		reg.RegisterWrapper(cveDackbox.Bucket, cveIndex.Wrapper{})
		reg.RegisterWrapper(deploymentDackbox.Bucket, deploymentIndex.Wrapper{})
		reg.RegisterWrapper(imageDackbox.Bucket, imageIndex.Wrapper{})
		reg.RegisterWrapper(imageComponentDackbox.Bucket, imageComponentIndex.Wrapper{})
		reg.RegisterWrapper(imageComponentEdgeDackbox.Bucket, imageComponentEdgeIndex.Wrapper{})
		reg.RegisterWrapper(imageCVEEdgeDackbox.Bucket, imageCVEEdgeIndex.Wrapper{})
		reg.RegisterWrapper(nodeDackbox.Bucket, nodeIndex.Wrapper{})
		reg.RegisterWrapper(nodeComponentEdgeDackbox.Bucket, nodeComponentEdgeIndex.Wrapper{})
		s.nodeStore = nodeDataStore.GetTestRocksBleveDataStore(t, s.rocksEngine, s.bleveIndex, s.dacky, s.keyFence)
		s.imageStore = imageDataStore.GetTestRocksBleveDataStore(t, s.rocksEngine, s.bleveIndex, s.dacky, s.keyFence)
		s.deploymentStore = deploymentDataStore.GetTestRocksBleveDataStore(t, s.rocksEngine, s.bleveIndex, s.dacky, s.keyFence)
		s.namespaceStore = namespaceDataStore.GetTestRocksBleveDataStore(t, s.rocksEngine, s.bleveIndex, s.dacky, s.keyFence)
	}
	return s, nil
}
