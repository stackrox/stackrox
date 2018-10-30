package store

import (
	"os"
	"testing"

	"github.com/boltdb/bolt"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/dberrors"
	"github.com/stretchr/testify/suite"
)

func TestDeploymentStore(t *testing.T) {
	suite.Run(t, new(DeploymentStoreTestSuite))
}

type DeploymentStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store Store
}

func (suite *DeploymentStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp(suite.T().Name() + ".db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store, err = New(db)
	suite.Require().NoError(err)
}

func (suite *DeploymentStoreTestSuite) TeardownSuite() {
	suite.db.Close()
	os.Remove(suite.db.Path())
}

func (suite *DeploymentStoreTestSuite) verifyDeploymentsAre(store Store, deployments ...*v1.Deployment) {
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
		suite.Equal(&v1.ListDeployment{
			Id:        d.GetId(),
			Name:      d.GetName(),
			UpdatedAt: d.GetUpdatedAt(),
			Priority:  d.GetPriority(),
		}, gotList)
	}

	// Test Count
	count, err := store.CountDeployments()
	suite.NoError(err)
	suite.Equal(len(deployments), count)
}

func (suite *DeploymentStoreTestSuite) TestDeployments() {
	deployments := []*v1.Deployment{
		{
			Id:        "fooID",
			Name:      "foo",
			Version:   "100",
			Type:      "Replicated",
			UpdatedAt: ptypes.TimestampNow(),
			Risk:      &v1.Risk{Score: 10},
		},
		{
			Id:        "barID",
			Name:      "bar",
			Version:   "400",
			Type:      "Global",
			UpdatedAt: ptypes.TimestampNow(),
			Risk:      &v1.Risk{Score: 9},
		},
	}

	// Test Add
	for _, d := range deployments {
		err := suite.store.UpdateDeployment(d)
		suite.Require().Error(err)
		suite.Equal(dberrors.ErrNotFound{Type: "Deployment", ID: d.GetId()}, err)
		suite.NoError(suite.store.UpsertDeployment(d))
		// Update should be idempotent
		suite.NoError(suite.store.UpdateDeployment(d))
	}

	// We want to make sure the priorities are set by the ranker.
	// We explicitly add them here for the comparisons later.
	for i, d := range deployments {
		d.Priority = int64(i + 1)
	}

	suite.verifyDeploymentsAre(suite.store, deployments...)

	// Test Update
	for _, d := range deployments {
		d.UpdatedAt = ptypes.TimestampNow()
		d.Version += "0"
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
