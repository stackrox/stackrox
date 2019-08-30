package badger

import (
	"testing"

	"github.com/dgraph-io/badger"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/deployment/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestDeploymentStore(t *testing.T) {
	suite.Run(t, new(DeploymentStoreTestSuite))
}

type DeploymentStoreTestSuite struct {
	suite.Suite

	db  *badger.DB
	dir string

	store store.Store
}

func (suite *DeploymentStoreTestSuite) SetupSuite() {
	db, dir, err := badgerhelper.NewTemp(suite.T().Name() + ".db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.dir = dir
	suite.store, err = New(db)
	suite.Require().NoError(err)
}

func (suite *DeploymentStoreTestSuite) TearDownSuite() {
	testutils.TearDownBadger(suite.db, suite.dir)
}

func (suite *DeploymentStoreTestSuite) verifyDeploymentsAre(store store.Store, deployments ...*storage.Deployment) {
	for _, d := range deployments {
		// Test retrieval of full objects
		got, exists, err := store.GetDeployment(d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(d, got)

		// Test retrieval of list objects
		gotList, exists, err := store.ListDeployment(d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(&storage.ListDeployment{
			Id:      d.GetId(),
			Name:    d.GetName(),
			Created: d.GetCreated(),
		}, gotList)
	}

	// Test Count
	count, err := store.CountDeployments()
	suite.NoError(err)
	suite.Equal(len(deployments), count)
}

func (suite *DeploymentStoreTestSuite) TestDeployments() {
	deployments := []*storage.Deployment{
		{
			Id:      "fooID",
			Name:    "foo",
			Type:    "Replicated",
			Created: ptypes.TimestampNow(),
		},
		{
			Id:      "barID",
			Name:    "bar",
			Type:    "Global",
			Created: ptypes.TimestampNow(),
		},
	}

	// Test Add
	for _, d := range deployments {
		err := suite.store.UpdateDeployment(d)
		suite.Require().Error(err)
		suite.Error(err)
		suite.NoError(suite.store.UpsertDeployment(d))
		// Update should be idempotent
		suite.NoError(suite.store.UpdateDeployment(d))
	}

	suite.verifyDeploymentsAre(suite.store, deployments...)

	// Test Update
	for _, d := range deployments {
		d.Created = ptypes.TimestampNow()
		d.Name += "-ext"
		suite.NoError(suite.store.UpdateDeployment(d))
	}

	suite.verifyDeploymentsAre(suite.store, deployments...)

	// This verifies that things work as expected on restarts.
	newStore, err := New(suite.db)
	suite.Require().NoError(err)

	suite.verifyDeploymentsAre(newStore, deployments...)

	// Test Remove
	for _, d := range deployments {
		suite.NoError(suite.store.RemoveDeployment(d.GetId()))
	}

	suite.verifyDeploymentsAre(suite.store)

	newStore, err = New(suite.db)
	suite.Require().NoError(err)

	suite.verifyDeploymentsAre(newStore)
}
