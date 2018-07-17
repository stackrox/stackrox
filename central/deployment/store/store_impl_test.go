package store

import (
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/central/ranking"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/bolthelper"
	"github.com/boltdb/bolt"
	ptypes "github.com/gogo/protobuf/types"
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
	suite.store = New(db, ranking.NewRanker())
}

func (suite *DeploymentStoreTestSuite) TeardownSuite() {
	suite.db.Close()
	os.Remove(suite.db.Path())
}

func (suite *DeploymentStoreTestSuite) TestDeployments() {
	deployments := []*v1.Deployment{
		{
			Id:        "fooID",
			Name:      "foo",
			Version:   "100",
			Type:      "Replicated",
			UpdatedAt: ptypes.TimestampNow(),
			Priority:  1,
		},
		{
			Id:        "barID",
			Name:      "bar",
			Version:   "400",
			Type:      "Global",
			UpdatedAt: ptypes.TimestampNow(),
			Priority:  1,
		},
	}

	// Test Add
	for _, d := range deployments {
		suite.NoError(suite.store.AddDeployment(d))
	}

	for _, d := range deployments {
		// Test retrieval of full objects
		got, exists, err := suite.store.GetDeployment(d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, d)

		// Test retrieval of list objects
		gotList, exists, err := suite.store.ListDeployment(d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(d.GetName(), gotList.GetName())
	}

	// Test Update
	for _, d := range deployments {
		d.UpdatedAt = ptypes.TimestampNow()
		d.Version += "0"
	}

	for _, d := range deployments {
		d.Name += "-ext"
		suite.NoError(suite.store.UpdateDeployment(d))
	}

	for _, d := range deployments {
		got, exists, err := suite.store.GetDeployment(d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, d)

		listGot, exists, err := suite.store.ListDeployment(d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(listGot.GetName(), listGot.GetName())
	}

	// Test Count
	count, err := suite.store.CountDeployments()
	suite.NoError(err)
	suite.Equal(len(deployments), count)

	// Test Remove
	for _, d := range deployments {
		suite.NoError(suite.store.RemoveDeployment(d.GetId()))
	}

	for _, d := range deployments {
		_, exists, err := suite.store.GetDeployment(d.GetId())
		suite.NoError(err)
		suite.False(exists)

		_, exists, err = suite.store.ListDeployment(d.GetId())
		suite.NoError(err)
		suite.False(exists)
	}

	// Test tombstones are set
	tombstoned, err := suite.store.GetTombstonedDeployments()
	for _, d := range tombstoned {
		suite.NoError(err)
		suite.NotNil(d.GetTombstone())
	}
}
