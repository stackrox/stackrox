package dackbox

import (
	"context"
	"testing"

	ptypes "github.com/gogo/protobuf/types"
	clusterDackBox "github.com/stackrox/rox/central/cluster/dackbox"
	namespaceDackBox "github.com/stackrox/rox/central/namespace/dackbox"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

func TestDeploymentStore(t *testing.T) {
	suite.Run(t, new(DeploymentStoreTestSuite))
}

type DeploymentStoreTestSuite struct {
	suite.Suite

	db    *rocksdb.RocksDB
	dacky *dackbox.DackBox

	store *StoreImpl

	ctx context.Context
}

func (suite *DeploymentStoreTestSuite) SetupSuite() {
	var err error
	suite.db, err = rocksdb.NewTemp("reference")
	if err != nil {
		suite.FailNowf("failed to create DB: %+v", err.Error())
	}
	suite.dacky, err = dackbox.NewRocksDBDackBox(suite.db, nil, []byte("graph"), []byte("dirty"), []byte("valid"))
	if err != nil {
		suite.FailNowf("failed to create dackbox: %+v", err.Error())
	}
	suite.store = New(suite.dacky, concurrency.NewKeyFence())
	suite.ctx = sac.WithAllAccess(context.Background())
}

func (suite *DeploymentStoreTestSuite) TearDownSuite() {
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *DeploymentStoreTestSuite) verifyDeploymentsAre(store *StoreImpl, deployments ...*storage.Deployment) {
	for _, d := range deployments {
		// Test retrieval of full objects
		got, exists, err := store.Get(suite.ctx, d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(d, got)

		// Test retrieval of list objects
		gotList, exists, err := store.GetListDeployment(suite.ctx, d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(&storage.ListDeployment{
			Id:        d.GetId(),
			Name:      d.GetName(),
			ClusterId: d.GetClusterId(),
			Namespace: d.GetNamespace(),
			Created:   d.GetCreated(),
		}, gotList)
	}

	// Test Count
	count, err := store.Count(suite.ctx)
	suite.NoError(err)
	suite.Equal(len(deployments), count)
}

func (suite *DeploymentStoreTestSuite) TestDeployments() {
	deployments1 := []*storage.Deployment{
		{
			Id:          "fooID",
			Name:        "foo",
			ClusterId:   "c1",
			NamespaceId: "n1",
			Namespace:   "n1",
			Type:        "Replicated",
			Created:     ptypes.TimestampNow(),
		},
		{
			Id:          "barID",
			Name:        "bar",
			ClusterId:   "c1",
			NamespaceId: "n2",
			Namespace:   "n2",
			Type:        "Global",
			Created:     ptypes.TimestampNow(),
		},
	}

	// Test Add
	for _, d := range deployments1 {
		suite.NoError(suite.store.Upsert(suite.ctx, d))
	}

	suite.verifyDeploymentsAre(suite.store, deployments1...)

	// This verifies that things work as expected on restarts.
	newStore := New(suite.dacky, concurrency.NewKeyFence())

	suite.verifyDeploymentsAre(newStore, deployments1...)

	deployments2 := []*storage.Deployment{
		{
			Id:          "fooID",
			Name:        "foo",
			ClusterId:   "c2",
			NamespaceId: "n1",
			Namespace:   "n1",
			Type:        "Replicated",
			Created:     ptypes.TimestampNow(),
		},
		{
			Id:          "barID",
			Name:        "bar",
			ClusterId:   "c1",
			NamespaceId: "n2",
			Namespace:   "n2",
			Type:        "Global",
			Created:     ptypes.TimestampNow(),
		},
	}

	for _, d := range deployments2 {
		suite.NoError(suite.store.Upsert(suite.ctx, d))
	}

	suite.verifyDeploymentsAre(newStore, deployments2...)

	// Check that when the cluster is updated for the deployments, old cluster info is overwritten.
	gView1 := suite.dacky.NewGraphView()
	defer gView1.Discard()
	suite.Equal([][]byte{clusterDackBox.BucketHandler.GetKey("c2")}, gView1.GetRefsTo(namespaceDackBox.BucketHandler.GetKey("n1")))
	suite.Equal([][]byte{clusterDackBox.BucketHandler.GetKey("c1")}, gView1.GetRefsTo(namespaceDackBox.BucketHandler.GetKey("n2")))

	// Test Remove
	for _, d := range deployments2 {
		suite.NoError(suite.store.Delete(suite.ctx, d.GetId()))
	}
	suite.verifyDeploymentsAre(suite.store)

	// Verify that when all deployments are removed, the namespace and cluster mappings for those deployments are also removed.
	gView2 := suite.dacky.NewGraphView()
	defer gView2.Discard()
	suite.Equal(0, gView2.CountRefsFrom(clusterDackBox.BucketHandler.GetKey("c1")))
	suite.Equal(0, gView2.CountRefsFrom(clusterDackBox.BucketHandler.GetKey("c1")))
	suite.Equal(0, gView2.CountRefsTo(namespaceDackBox.BucketHandler.GetKey("n1")))
	suite.Equal(0, gView2.CountRefsTo(namespaceDackBox.BucketHandler.GetKey("n2")))

	newStore = New(suite.dacky, concurrency.NewKeyFence())

	suite.verifyDeploymentsAre(newStore)
}
