package bolt

import (
	"context"
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/bolthelper"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestNetworkPolicyStore(t *testing.T) {
	suite.Run(t, new(NetworkPolicyStoreTestSuite))
}

type NetworkPolicyStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store *storeImpl
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

	ctx := context.Background()

	// Test Add
	for _, np := range networkPolicies {
		suite.NoError(suite.store.Upsert(ctx, np))
	}

	for _, d := range networkPolicies {
		got, exists, err := suite.store.Get(ctx, d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, d)
	}

	var existingPols []*storage.NetworkPolicy
	err := suite.store.Walk(ctx, func(np *storage.NetworkPolicy) error {
		existingPols = append(existingPols, np)
		return nil
	})
	suite.NoError(err)
	suite.ElementsMatch(existingPols, networkPolicies)

	// Test Update
	for _, d := range networkPolicies {
		d.ClusterId += "1"
	}

	for _, d := range networkPolicies {
		suite.NoError(suite.store.Upsert(ctx, d))
	}

	for _, d := range networkPolicies {
		got, exists, err := suite.store.Get(ctx, d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, d)
	}

	// Test Remove
	for _, d := range networkPolicies {
		suite.NoError(suite.store.Delete(ctx, d.GetId()))
	}

	for _, d := range networkPolicies {
		_, exists, err := suite.store.Get(ctx, d.GetId())
		suite.NoError(err)
		suite.False(exists)
	}
}
