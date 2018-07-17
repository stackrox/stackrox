package store

import (
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/bolthelper"
	"github.com/boltdb/bolt"
	"github.com/stretchr/testify/suite"
)

func TestDeploymentStore(t *testing.T) {
	suite.Run(t, new(NetworkPolicyStoreTestSuite))
}

type NetworkPolicyStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store Store
}

func (suite *NetworkPolicyStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp(suite.T().Name() + ".db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store = New(db)
}

func (suite *NetworkPolicyStoreTestSuite) TeardownSuite() {
	suite.db.Close()
	os.Remove(suite.db.Path())
}

func (suite *NetworkPolicyStoreTestSuite) TestNetworkPolicies() {
	networkPolicies := []*v1.NetworkPolicy{
		{
			Id:        "fooID",
			ClusterId: "1",
		},
		{
			Id:        "barID",
			ClusterId: "2",
		},
	}

	// Test Add
	for _, np := range networkPolicies {
		suite.NoError(suite.store.AddNetworkPolicy(np))
	}

	for _, d := range networkPolicies {
		got, exists, err := suite.store.GetNetworkPolicy(d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, d)
	}

	// Test Update
	for _, d := range networkPolicies {
		d.ClusterId += "1"
	}

	for _, d := range networkPolicies {
		suite.NoError(suite.store.UpdateNetworkPolicy(d))
	}

	for _, d := range networkPolicies {
		got, exists, err := suite.store.GetNetworkPolicy(d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, d)
	}

	// Test Count
	count, err := suite.store.CountNetworkPolicies()
	suite.NoError(err)
	suite.Equal(len(networkPolicies), count)

	// Test Remove
	for _, d := range networkPolicies {
		suite.NoError(suite.store.RemoveNetworkPolicy(d.GetId()))
	}

	for _, d := range networkPolicies {
		_, exists, err := suite.store.GetNetworkPolicy(d.GetId())
		suite.NoError(err)
		suite.False(exists)
	}
}
