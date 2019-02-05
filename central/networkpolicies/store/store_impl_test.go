package store

import (
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
	suite.NoError(suite.db.Close())
}

func (suite *NetworkPolicyStoreTestSuite) expectPolicies(req *v1.GetNetworkPoliciesRequest, wantPolicies ...*storage.NetworkPolicy) {
	gotPolicies, err := suite.store.GetNetworkPolicies(req)
	suite.Require().NoError(err)
	suite.ElementsMatch(gotPolicies, wantPolicies)
	gotCount, err := suite.store.CountMatchingNetworkPolicies(req)
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

	suite.expectPolicies(&v1.GetNetworkPoliciesRequest{}, networkPolicies...)
	suite.expectPolicies(&v1.GetNetworkPoliciesRequest{ClusterId: "1"}, networkPolicies[0])
	suite.expectPolicies(&v1.GetNetworkPoliciesRequest{ClusterId: "2"}, networkPolicies[1])
	suite.expectPolicies(&v1.GetNetworkPoliciesRequest{ClusterId: "INVALID"})
	suite.expectPolicies(&v1.GetNetworkPoliciesRequest{Namespace: "NS1"}, networkPolicies[0])
	suite.expectPolicies(&v1.GetNetworkPoliciesRequest{ClusterId: "1", Namespace: "INVALID"})
	suite.expectPolicies(&v1.GetNetworkPoliciesRequest{ClusterId: "INVALID", Namespace: "NS1"})
	suite.expectPolicies(&v1.GetNetworkPoliciesRequest{Namespace: "NS2"}, networkPolicies[1])
	suite.expectPolicies(&v1.GetNetworkPoliciesRequest{ClusterId: "2", Namespace: "NS2"}, networkPolicies[1])
	suite.expectPolicies(&v1.GetNetworkPoliciesRequest{ClusterId: "1", Namespace: "NS2"})

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
