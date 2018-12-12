package store

import (
	"os"
	"sort"
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stretchr/testify/suite"
)

func TestNetworkPolicyStore(t *testing.T) {
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

func (suite *NetworkPolicyStoreTestSuite) TearDownSuite() {
	suite.db.Close()
	os.Remove(suite.db.Path())
}

func (suite *NetworkPolicyStoreTestSuite) TestNetworkPolicies() {
	networkPolicies := []*storage.NetworkPolicy{
		{
			Id:        "1fooID",
			ClusterId: "1",
		},
		{
			Id:        "2barID",
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

	policies, err := suite.store.GetNetworkPolicies(&v1.GetNetworkPoliciesRequest{})
	suite.Require().NoError(err)
	suite.Len(policies, 2)
	sort.Slice(policies, func(i, j int) bool {
		return policies[i].GetId() < policies[j].GetId()
	})
	suite.Equal(policies, networkPolicies)

	policies, err = suite.store.GetNetworkPolicies(&v1.GetNetworkPoliciesRequest{ClusterId: "1"})
	suite.Require().NoError(err)
	suite.Len(policies, 1)
	suite.Equal(policies[0], networkPolicies[0])

	policies, err = suite.store.GetNetworkPolicies(&v1.GetNetworkPoliciesRequest{ClusterId: "2"})
	suite.Require().NoError(err)
	suite.Len(policies, 1)
	suite.Equal(policies[0], networkPolicies[1])

	policies, err = suite.store.GetNetworkPolicies(&v1.GetNetworkPoliciesRequest{ClusterId: "INVALID"})
	suite.Require().NoError(err)
	suite.Len(policies, 0)

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
