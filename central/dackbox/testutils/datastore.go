package testutils

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/jackc/pgx/v4/pgxpool"
	activeComponentDackbox "github.com/stackrox/rox/central/activecomponent/dackbox"
	activeComponentIndex "github.com/stackrox/rox/central/activecomponent/datastore/index"
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
	dackboxConcurrency "github.com/stackrox/rox/pkg/dackbox/concurrency"
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

// DackboxTestDataStore provides the interface to a utility for dackbox testing, with accessors to some internals,
// as well as test data injection and cleanup functions.
type DackboxTestDataStore interface {
	// Expose internal for the case other datastores would be needed for testing purposes
	GetPostgresPool() *pgxpool.Pool
	GetRocksEngine() *rocksPkg.RocksDB
	GetBleveIndex() bleve.Index
	GetDackbox() *dackbox.DackBox
	GetKeyFence() dackboxConcurrency.KeyFence
	GetIndexQ() queue.WaitableQueue
	// Data injection
	PushImageToVulnerabilitiesGraph(waitForIndexing bool) error
	PushNodeToVulnerabilitiesGraph(waitForIndexing bool) error
	// Data cleanup
	CleanImageToVulnerabilitiesGraph(waitForIndexing bool) error
	CleanNodeToVulnerabilitiesGraph(waitForIndexing bool) error
	// Post test cleanup (TearDown)
	Cleanup(t *testing.T) error
}

type dackboxTestDataStoreImpl struct {
	// Pool for postgres mode
	pgtestbase *pgtest.TestPostgres
	// Elements for rocksdb+bleve mode
	rocksEngine *rocksPkg.RocksDB
	bleveIndex  bleve.Index
	dacky       *dackbox.DackBox
	keyFence    dackboxConcurrency.KeyFence
	indexQ      queue.WaitableQueue

	// DataStores
	namespaceStore  namespaceDataStore.DataStore
	deploymentStore deploymentDataStore.DataStore
	imageStore      imageDataStore.DataStore
	nodeStore       nodeDataStore.DataStore

	storedNodes       []string
	storedImages      []string
	storedDeployments []string
	storedNamespaces  []string
}

func (s *dackboxTestDataStoreImpl) GetPostgresPool() *pgxpool.Pool {
	return s.pgtestbase.Pool
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

func (s *dackboxTestDataStoreImpl) GetKeyFence() dackboxConcurrency.KeyFence {
	return s.keyFence
}

func (s *dackboxTestDataStoreImpl) GetIndexQ() queue.WaitableQueue {
	return s.indexQ
}

// PushImageToVulnerabilitiesGraph inserts the namespace -> deployment -> image -> CVE graph defined
// in the dackbox fixture (see the comment at the top of the image section for more details).
// This function creates NamespaceA in Cluster1 and NamespaceB in Cluster2, then injects the SherlockHolmes
// and DoctorJekyll images, to finally bind them to their respective namespaces through the identically
// names deployments.
// Sherlock holmes is the deployment / image part from Cluster1 and NamespaceA.
// Dr Jekyll is the deployment / image part from Cluster2 and NamespaceB.
func (s *dackboxTestDataStoreImpl) PushImageToVulnerabilitiesGraph(waitForIndexing bool) (err error) {
	ctx := sac.WithAllAccess(context.Background())
	testNamespace1 := fixtures.GetNamespace(testconsts.Cluster1, testconsts.Cluster1, testconsts.NamespaceA)
	testNamespace2 := fixtures.GetNamespace(testconsts.Cluster2, testconsts.Cluster2, testconsts.NamespaceB)
	testImage1 := fixtures.GetImageSherlockHolmes1()
	testImage2 := fixtures.GetImageDoctorJekyll2()
	testDeployment1 := fixtures.GetDeploymentSherlockHolmes1(uuid.NewV4().String(), testNamespace1)
	testDeployment2 := fixtures.GetDeploymentDoctorJekyll2(uuid.NewV4().String(), testNamespace2)
	err = s.namespaceStore.AddNamespace(ctx, testNamespace1)
	if err != nil {
		return err
	}
	s.storedNamespaces = append(s.storedNamespaces, testNamespace1.GetId())
	err = s.namespaceStore.AddNamespace(ctx, testNamespace2)
	if err != nil {
		return err
	}
	s.storedNamespaces = append(s.storedNamespaces, testNamespace2.GetId())
	err = s.imageStore.UpsertImage(ctx, testImage1)
	if err != nil {
		return err
	}
	s.storedImages = append(s.storedImages, testImage1.GetId())
	err = s.imageStore.UpsertImage(ctx, testImage2)
	if err != nil {
		return err
	}
	s.storedImages = append(s.storedImages, testImage2.GetId())
	err = s.deploymentStore.UpsertDeployment(ctx, testDeployment1)
	if err != nil {
		return err
	}
	s.storedDeployments = append(s.storedDeployments, testDeployment1.GetId())
	err = s.deploymentStore.UpsertDeployment(ctx, testDeployment2)
	if err != nil {
		return err
	}
	s.storedDeployments = append(s.storedDeployments, testDeployment2.GetId())
	if waitForIndexing {
		s.waitForIndexing()
	}
	return nil
}

// PushNodeToVulnerabilitiesGraph inserts the node -> CVE graph defined
// in the dackbox fixture (see the comment at the top of the image section for more details).
// Sherlock holmes is the node part from Cluster1.
// Dr Jekyll is the node part from Cluster2.
func (s *dackboxTestDataStoreImpl) PushNodeToVulnerabilitiesGraph(waitForIndexing bool) (err error) {
	ctx := sac.WithAllAccess(context.Background())
	testNode1 := fixtures.GetScopedNode1(uuid.NewV4().String(), testconsts.Cluster1)
	testNode2 := fixtures.GetScopedNode2(uuid.NewV4().String(), testconsts.Cluster2)
	err = s.nodeStore.UpsertNode(ctx, testNode1)
	if err != nil {
		return err
	}
	s.storedNodes = append(s.storedNodes, testNode1.GetId())
	err = s.nodeStore.UpsertNode(ctx, testNode2)
	if err != nil {
		return err
	}
	s.storedNodes = append(s.storedNodes, testNode2.GetId())
	if waitForIndexing {
		s.waitForIndexing()
	}
	return nil
}

// CleanImageToVulnerabilitiesGraph removes from database the data injected by PushImageToVulnerabilitiesGraph.
func (s *dackboxTestDataStoreImpl) CleanImageToVulnerabilitiesGraph(waitForIndexing bool) (err error) {
	ctx := sac.WithAllAccess(context.Background())
	storedDeployments := s.storedDeployments
	for _, deploymentID := range storedDeployments {
		deployment, found, err := s.deploymentStore.GetDeployment(ctx, deploymentID)
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		err = s.deploymentStore.RemoveDeployment(ctx, deployment.GetClusterId(), deploymentID)
		if err != nil {
			return err
		}
	}
	s.storedDeployments = s.storedDeployments[:0]
	storedImages := s.storedImages
	for _, imageID := range storedImages {
		err = s.imageStore.DeleteImages(ctx, imageID)
		if err != nil {
			return err
		}
	}
	s.storedImages = s.storedImages[:0]
	storedNamespaces := s.storedNamespaces
	for _, namespaceID := range storedNamespaces {
		err := s.namespaceStore.RemoveNamespace(ctx, namespaceID)
		if err != nil {
			return err
		}
	}
	s.storedNamespaces = s.storedNamespaces[:0]
	if waitForIndexing {
		s.waitForIndexing()
	}
	return nil
}

// CleanNodeToVulnerabilitiesGraph removes from database the data injected by PushNodeToVulnerabilitiesGraph.
func (s *dackboxTestDataStoreImpl) CleanNodeToVulnerabilitiesGraph(waitForIndexing bool) (err error) {
	ctx := sac.WithAllAccess(context.Background())
	storedNodes := s.storedNodes
	for _, nodeID := range storedNodes {
		err = s.nodeStore.DeleteNodes(ctx, nodeID)
		if err != nil {
			return err
		}
	}
	s.storedNodes = s.storedNodes[:0]
	if waitForIndexing {
		s.waitForIndexing()
	}
	return nil
}

func (s *dackboxTestDataStoreImpl) waitForIndexing() {
	if !features.PostgresDatastore.Enabled() {
		indexingCompleted := concurrency.NewSignal()
		s.indexQ.PushSignal(&indexingCompleted)
		<-indexingCompleted.Done()
	}
}

func (s *dackboxTestDataStoreImpl) Cleanup(t *testing.T) (err error) {
	if features.PostgresDatastore.Enabled() {
		s.pgtestbase.Teardown(t)
	} else {
		s.waitForIndexing()
		err = s.bleveIndex.Close()
		if err != nil {
			return err
		}
		err = rocksPkg.CloseAndRemove(s.rocksEngine)
		if err != nil {
			return err
		}
	}
	return nil
}

// NewDackboxTestDataStore provides a utility for dackbox storage testing, which contains a set of connected
// dackbox datastores, as well as a set of functions to inject and cleanup data.
func NewDackboxTestDataStore(t *testing.T) (DackboxTestDataStore, error) {
	var err error
	s := &dackboxTestDataStoreImpl{}
	if features.PostgresDatastore.Enabled() {
		s.pgtestbase = pgtest.ForT(t)
		s.nodeStore, err = nodeDataStore.GetTestPostgresDataStore(t, s.GetPostgresPool())
		if err != nil {
			return nil, err
		}
		s.imageStore, err = imageDataStore.GetTestPostgresDataStore(t, s.GetPostgresPool())
		if err != nil {
			return nil, err
		}
		s.deploymentStore, err = deploymentDataStore.GetTestPostgresDataStore(t, s.GetPostgresPool())
		if err != nil {
			return nil, err
		}
		s.namespaceStore, err = namespaceDataStore.GetTestPostgresDataStore(t, s.GetPostgresPool())
		if err != nil {
			return nil, err
		}
	} else {
		s.rocksEngine, err = rocksPkg.NewTemp("dackboxtest")
		if err != nil {
			return nil, err
		}
		s.bleveIndex, err = globalindex.MemOnlyIndex()
		if err != nil {
			return nil, err
		}
		s.keyFence = dackboxConcurrency.NewKeyFence()
		s.indexQ = queue.NewWaitableQueue()
		s.dacky, err = dackbox.NewRocksDBDackBox(s.rocksEngine, s.indexQ, []byte("graph"), []byte("dirty"), []byte("valid"))
		if err != nil {
			return nil, err
		}
		reg := indexer.NewWrapperRegistry()
		indexer.NewLazy(s.indexQ, reg, s.bleveIndex, s.dacky.AckIndexed).Start()
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
		s.nodeStore, err = nodeDataStore.GetTestRocksBleveDataStore(t, s.rocksEngine, s.bleveIndex, s.dacky, s.keyFence)
		if err != nil {
			return nil, err
		}
		s.imageStore, err = imageDataStore.GetTestRocksBleveDataStore(t, s.rocksEngine, s.bleveIndex, s.dacky, s.keyFence)
		if err != nil {
			return nil, err
		}
		s.deploymentStore, err = deploymentDataStore.GetTestRocksBleveDataStore(t, s.rocksEngine, s.bleveIndex, s.dacky, s.keyFence)
		if err != nil {
			return nil, err
		}
		s.namespaceStore, err = namespaceDataStore.GetTestRocksBleveDataStore(t, s.rocksEngine, s.bleveIndex, s.dacky, s.keyFence)
		if err != nil {
			return nil, err
		}
	}
	s.storedDeployments = make([]string, 0)
	s.storedNamespaces = make([]string, 0)
	s.storedImages = make([]string, 0)
	s.storedNodes = make([]string, 0)

	return s, nil
}
