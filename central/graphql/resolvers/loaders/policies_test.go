package loaders

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/policy/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	policy1 = "policy1"
	policy2 = "policy2"
	policy3 = "policy3"
)

func TestPolicyLoader(t *testing.T) {
	suite.Run(t, new(PolicyLoaderTestSuite))
}

type PolicyLoaderTestSuite struct {
	suite.Suite

	ctx context.Context

	mockCtrl      *gomock.Controller
	mockDataStore *mocks.MockDataStore
}

func (suite *PolicyLoaderTestSuite) SetupTest() {
	suite.ctx = context.Background()

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockDataStore = mocks.NewMockDataStore(suite.mockCtrl)
}

func (suite *PolicyLoaderTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PolicyLoaderTestSuite) TestFromID() {
	// Create a loader with some reloaded policies.
	loader := policyLoaderImpl{
		loaded: map[string]*storage.Policy{
			"policy1": {Id: policy1},
			"policy2": {Id: policy2},
		},
		policyDS: suite.mockDataStore,
	}

	// Get a preloaded policy from id.
	policy, err := loader.FromID(suite.ctx, policy1)
	suite.NoError(err)
	suite.Equal(loader.loaded[policy1], policy)

	// Get a non-preloaded policy from id.
	thirdPolicy := &storage.Policy{Id: policy3}
	suite.mockDataStore.EXPECT().
		SearchRawPolicies(suite.ctx, search.NewQueryBuilder().AddDocIDs(policy3).ProtoQuery()).
		Return([]*storage.Policy{thirdPolicy}, nil)

	policy, err = loader.FromID(suite.ctx, policy3)
	suite.NoError(err)
	suite.Equal(thirdPolicy, policy)

	// Above call should now be preloaded.
	policy, err = loader.FromID(suite.ctx, policy3)
	suite.NoError(err)
	suite.Equal(loader.loaded[policy3], policy)
}

func (suite *PolicyLoaderTestSuite) TestFromIDs() {
	// Create a loader with some reloaded policies.
	loader := policyLoaderImpl{
		loaded: map[string]*storage.Policy{
			"policy1": {Id: policy1},
			"policy2": {Id: policy2},
		},
		policyDS: suite.mockDataStore,
	}

	// Get a preloaded policy from id.
	policies, err := loader.FromIDs(suite.ctx, []string{policy1, policy2})
	suite.NoError(err)
	suite.Equal([]*storage.Policy{
		loader.loaded[policy1],
		loader.loaded[policy2],
	}, policies)

	// Get a non-preloaded policy from id.
	thirdPolicy := &storage.Policy{Id: "policy3"}
	suite.mockDataStore.EXPECT().
		SearchRawPolicies(suite.ctx, search.NewQueryBuilder().AddDocIDs(policy3).ProtoQuery()).
		Return([]*storage.Policy{thirdPolicy}, nil)

	policies, err = loader.FromIDs(suite.ctx, []string{policy1, policy2, policy3})
	suite.NoError(err)
	suite.Equal([]*storage.Policy{
		loader.loaded[policy1],
		loader.loaded[policy2],
		thirdPolicy,
	}, policies)

	// Above call should now be preloaded.
	policies, err = loader.FromIDs(suite.ctx, []string{policy1, policy2, policy3})
	suite.NoError(err)
	suite.Equal([]*storage.Policy{
		loader.loaded[policy1],
		loader.loaded[policy2],
		loader.loaded[policy3],
	}, policies)
}

func (suite *PolicyLoaderTestSuite) TestFromQuery() {
	// Create a loader with some reloaded policies.
	loader := policyLoaderImpl{
		loaded: map[string]*storage.Policy{
			"policy1": {Id: policy1},
			"policy2": {Id: policy2},
		},
		policyDS: suite.mockDataStore,
	}
	query := &v1.Query{}

	// Get a preloaded policy from id.
	results := []search.Result{
		{
			ID: policy1,
		},
		{
			ID: policy2,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	policies, err := loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.Policy{
		loader.loaded[policy1],
		loader.loaded[policy2],
	}, policies)

	// Get a non-preloaded policy from id.
	results = []search.Result{
		{
			ID: policy1,
		},
		{
			ID: policy2,
		},
		{
			ID: policy3,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	thirdPolicy := &storage.Policy{Id: "policy3"}
	suite.mockDataStore.EXPECT().
		SearchRawPolicies(suite.ctx, search.NewQueryBuilder().AddDocIDs(policy3).ProtoQuery()).
		Return([]*storage.Policy{thirdPolicy}, nil)

	policies, err = loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.Policy{
		loader.loaded[policy1],
		loader.loaded[policy2],
		thirdPolicy,
	}, policies)

	// Above call should now be pre-loaded.
	results = []search.Result{
		{
			ID: policy1,
		},
		{
			ID: policy2,
		},
		{
			ID: policy3,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	policies, err = loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.Policy{
		loader.loaded[policy1],
		loader.loaded[policy2],
		loader.loaded[policy3],
	}, policies)
}
