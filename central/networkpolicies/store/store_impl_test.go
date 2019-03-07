package store

import (
	"testing"

	bolt "github.com/etcd-io/bbolt"
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
	suite.NoError(suite.db.Close())
}

func (suite *NetworkPolicyStoreTestSuite) expectPolicies(clusterID, namespace string, wantPolicies ...*storage.NetworkPolicy) {
	gotPolicies, err := suite.store.GetNetworkPolicies(clusterID, namespace)
	suite.Require().NoError(err)
	suite.ElementsMatch(gotPolicies, wantPolicies)
	gotCount, err := suite.store.CountMatchingNetworkPolicies(clusterID, namespace)
	suite.Require().NoError(err)
	suite.Equal(len(wantPolicies), gotCount)
}

func (suite *NetworkPolicyStoreTestSuite) TestNetworkPolicies() {
	networkPolicies := []*storage.NetworkPolicy{
		{
			Id:        "1fooID",
			ClusterId: "1",
			Namespace: "NS1",
		},
		{
			Id:        "2barID",
			ClusterId: "2",
			Namespace: "NS2",
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

	suite.expectPolicies("", "", networkPolicies...)
	suite.expectPolicies("1", "", networkPolicies[0])
	suite.expectPolicies("2", "", networkPolicies[1])
	suite.expectPolicies("INVALID", "")
	suite.expectPolicies("", "NS1", networkPolicies[0])
	suite.expectPolicies("1", "INVALID")
	suite.expectPolicies("INVALID", "NS1")
	suite.expectPolicies("", "NS2", networkPolicies[1])
	suite.expectPolicies("2", "NS2", networkPolicies[1])
	suite.expectPolicies("1", "NS2")

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
