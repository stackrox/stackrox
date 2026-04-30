package datastore

import (
	"context"
	"testing"

	dsMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestActiveStateDatastoreSuite(t *testing.T) {
	suite.Run(t, new(ActiveStateDatastoreTestSuite))
}

type ActiveStateDatastoreTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
	inner    *dsMocks.MockDataStore
	ds       DataStore
	ctx      context.Context
}

func (s *ActiveStateDatastoreTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.inner = dsMocks.NewMockDataStore(s.mockCtrl)
	s.ds = NewActiveStateDatastore(s.inner)
	s.ctx = context.Background()
}

func (s *ActiveStateDatastoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

// --- Query-based methods inject the active filter ---

func (s *ActiveStateDatastoreTestSuite) TestSearchInjectsActiveFilter() {
	s.T().Setenv(features.DeploymentSoftDeletion.EnvVar(), "true")
	testutils.MustUpdateFeature(s.T(), features.DeploymentSoftDeletion, true)

	baseQuery := pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.DeploymentName, "nginx").ProtoQuery()
	s.inner.EXPECT().Search(s.ctx, gomock.Any()).DoAndReturn(
		func(_ context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
			assertQueryContainsActiveFilter(s.T(), q)
			return nil, nil
		},
	)
	_, _ = s.ds.Search(s.ctx, baseQuery)
}

func (s *ActiveStateDatastoreTestSuite) TestCountInjectsActiveFilter() {
	s.T().Setenv(features.DeploymentSoftDeletion.EnvVar(), "true")
	testutils.MustUpdateFeature(s.T(), features.DeploymentSoftDeletion, true)

	s.inner.EXPECT().Count(s.ctx, gomock.Any()).DoAndReturn(
		func(_ context.Context, q *v1.Query) (int, error) {
			assertQueryContainsActiveFilter(s.T(), q)
			return 0, nil
		},
	)
	_, _ = s.ds.Count(s.ctx, pkgSearch.EmptyQuery())
}

func (s *ActiveStateDatastoreTestSuite) TestWalkByQueryInjectsActiveFilter() {
	s.T().Setenv(features.DeploymentSoftDeletion.EnvVar(), "true")
	testutils.MustUpdateFeature(s.T(), features.DeploymentSoftDeletion, true)

	s.inner.EXPECT().WalkByQuery(s.ctx, gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, q *v1.Query, _ func(*storage.Deployment) error) error {
			assertQueryContainsActiveFilter(s.T(), q)
			return nil
		},
	)
	_ = s.ds.WalkByQuery(s.ctx, pkgSearch.EmptyQuery(), func(_ *storage.Deployment) error { return nil })
}

// --- ID-based methods filter non-active deployments ---

func (s *ActiveStateDatastoreTestSuite) TestGetDeploymentFiltersDeleted() {
	s.T().Setenv(features.DeploymentSoftDeletion.EnvVar(), "true")
	testutils.MustUpdateFeature(s.T(), features.DeploymentSoftDeletion, true)

	deleted := &storage.Deployment{
		Id:    "d1",
		State: storage.DeploymentState_DEPLOYMENT_STATE_DELETED,
	}
	s.inner.EXPECT().GetDeployment(s.ctx, "d1").Return(deleted, true, nil)

	result, found, err := s.ds.GetDeployment(s.ctx, "d1")
	require.NoError(s.T(), err)
	assert.False(s.T(), found)
	assert.Nil(s.T(), result)
}

func (s *ActiveStateDatastoreTestSuite) TestGetDeploymentReturnsActive() {
	s.T().Setenv(features.DeploymentSoftDeletion.EnvVar(), "true")
	testutils.MustUpdateFeature(s.T(), features.DeploymentSoftDeletion, true)

	active := &storage.Deployment{
		Id:    "d1",
		State: storage.DeploymentState_DEPLOYMENT_STATE_ACTIVE,
	}
	s.inner.EXPECT().GetDeployment(s.ctx, "d1").Return(active, true, nil)

	result, found, err := s.ds.GetDeployment(s.ctx, "d1")
	require.NoError(s.T(), err)
	assert.True(s.T(), found)
	assert.Equal(s.T(), "d1", result.GetId())
}

func (s *ActiveStateDatastoreTestSuite) TestGetDeploymentNotFound() {
	s.T().Setenv(features.DeploymentSoftDeletion.EnvVar(), "true")
	testutils.MustUpdateFeature(s.T(), features.DeploymentSoftDeletion, true)

	s.inner.EXPECT().GetDeployment(s.ctx, "d1").Return(nil, false, nil)

	result, found, err := s.ds.GetDeployment(s.ctx, "d1")
	require.NoError(s.T(), err)
	assert.False(s.T(), found)
	assert.Nil(s.T(), result)
}

func (s *ActiveStateDatastoreTestSuite) TestGetDeploymentsFiltersDeleted() {
	s.T().Setenv(features.DeploymentSoftDeletion.EnvVar(), "true")
	testutils.MustUpdateFeature(s.T(), features.DeploymentSoftDeletion, true)

	deployments := []*storage.Deployment{
		{Id: "d1", State: storage.DeploymentState_DEPLOYMENT_STATE_ACTIVE},
		{Id: "d2", State: storage.DeploymentState_DEPLOYMENT_STATE_DELETED},
		{Id: "d3", State: storage.DeploymentState_DEPLOYMENT_STATE_ACTIVE},
	}
	s.inner.EXPECT().GetDeployments(s.ctx, []string{"d1", "d2", "d3"}).Return(deployments, nil)

	result, err := s.ds.GetDeployments(s.ctx, []string{"d1", "d2", "d3"})
	require.NoError(s.T(), err)
	require.Len(s.T(), result, 2)
	assert.Equal(s.T(), "d1", result[0].GetId())
	assert.Equal(s.T(), "d3", result[1].GetId())
}

func (s *ActiveStateDatastoreTestSuite) TestListDeploymentFiltersDeleted() {
	s.T().Setenv(features.DeploymentSoftDeletion.EnvVar(), "true")
	testutils.MustUpdateFeature(s.T(), features.DeploymentSoftDeletion, true)

	deleted := &storage.ListDeployment{
		Id:    "d1",
		State: storage.DeploymentState_DEPLOYMENT_STATE_DELETED,
	}
	s.inner.EXPECT().ListDeployment(s.ctx, "d1").Return(deleted, true, nil)

	result, found, err := s.ds.ListDeployment(s.ctx, "d1")
	require.NoError(s.T(), err)
	assert.False(s.T(), found)
	assert.Nil(s.T(), result)
}

// --- Feature flag disabled: no filtering ---

func (s *ActiveStateDatastoreTestSuite) TestGetDeploymentNoFilterWhenFlagDisabled() {
	s.T().Setenv(features.DeploymentSoftDeletion.EnvVar(), "false")
	testutils.MustUpdateFeature(s.T(), features.DeploymentSoftDeletion, false)

	// Even a deployment with DEPLOYMENT_STATE_DELETED should be returned when the flag is off.
	deleted := &storage.Deployment{
		Id:    "d1",
		State: storage.DeploymentState_DEPLOYMENT_STATE_DELETED,
	}
	s.inner.EXPECT().GetDeployment(s.ctx, "d1").Return(deleted, true, nil)

	result, found, err := s.ds.GetDeployment(s.ctx, "d1")
	require.NoError(s.T(), err)
	assert.True(s.T(), found)
	assert.Equal(s.T(), "d1", result.GetId())
}

func (s *ActiveStateDatastoreTestSuite) TestSearchNoFilterWhenFlagDisabled() {
	s.T().Setenv(features.DeploymentSoftDeletion.EnvVar(), "false")
	testutils.MustUpdateFeature(s.T(), features.DeploymentSoftDeletion, false)

	// When the flag is disabled, ActiveDeploymentsQuery returns an empty query,
	// so the conjunction should not add a state filter.
	s.inner.EXPECT().Search(s.ctx, gomock.Any()).DoAndReturn(
		func(_ context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
			assertQueryDoesNotContainActiveFilter(s.T(), q)
			return nil, nil
		},
	)
	_, _ = s.ds.Search(s.ctx, pkgSearch.EmptyQuery())
}

// --- Pass-through methods ---

func (s *ActiveStateDatastoreTestSuite) TestUpsertDelegatesDirectly() {
	dep := &storage.Deployment{Id: "d1"}
	s.inner.EXPECT().UpsertDeployment(s.ctx, dep).Return(nil)
	err := s.ds.UpsertDeployment(s.ctx, dep)
	assert.NoError(s.T(), err)
}

func (s *ActiveStateDatastoreTestSuite) TestRemoveDelegatesDirectly() {
	s.inner.EXPECT().RemoveDeployment(s.ctx, "cluster1", "d1").Return(nil)
	err := s.ds.RemoveDeployment(s.ctx, "cluster1", "d1")
	assert.NoError(s.T(), err)
}

func (s *ActiveStateDatastoreTestSuite) TestGetImagesForDeploymentDelegatesDirectly() {
	dep := &storage.Deployment{Id: "d1"}
	s.inner.EXPECT().GetImagesForDeployment(s.ctx, dep).Return(nil, nil)
	_, err := s.ds.GetImagesForDeployment(s.ctx, dep)
	assert.NoError(s.T(), err)
}

// --- Helpers ---

// assertQueryContainsActiveFilter checks that the query contains a match
// for the DeploymentState field with DEPLOYMENT_STATE_ACTIVE.
func assertQueryContainsActiveFilter(t *testing.T, q *v1.Query) {
	t.Helper()
	serialized := q.String()
	assert.Contains(t, serialized, storage.DeploymentState_DEPLOYMENT_STATE_ACTIVE.String(),
		"expected query to contain DEPLOYMENT_STATE_ACTIVE filter")
}

// assertQueryDoesNotContainActiveFilter checks that the query does not contain
// a match for the DeploymentState field.
func assertQueryDoesNotContainActiveFilter(t *testing.T, q *v1.Query) {
	t.Helper()
	serialized := q.String()
	assert.NotContains(t, serialized, storage.DeploymentState_DEPLOYMENT_STATE_ACTIVE.String(),
		"expected query to NOT contain active-deployment state filter")
}
