package service

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	namespaceMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	roleMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestServiceImpl(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ServiceTestSuite))
}

type ServiceTestSuite struct {
	suite.Suite

	envIsolator    *envisolator.EnvIsolator
	requestContext context.Context

	mockCtrl *gomock.Controller

	mockRoles      *roleMocks.MockDataStore
	mockClusters   *clusterMocks.MockDataStore
	mockNamespaces *namespaceMocks.MockDataStore

	svc Service
}

func (s *ServiceTestSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.requestContext = context.Background()

	s.mockCtrl = gomock.NewController(s.T())

	s.mockRoles = roleMocks.NewMockDataStore(s.mockCtrl)
	s.mockClusters = clusterMocks.NewMockDataStore(s.mockCtrl)
	s.mockNamespaces = namespaceMocks.NewMockDataStore(s.mockCtrl)

	s.svc = New(s.mockRoles, s.mockClusters, s.mockNamespaces)
}

func (s *ServiceTestSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
	s.mockCtrl.Finish()
}

func (s *ServiceTestSuite) TestAccessScopeAPIDisabledByDefault() {
	s.envIsolator.Setenv(features.ScopedAccessControl.EnvVar(), "false")
	s.False(features.ScopedAccessControl.Enabled())

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())

	_, err := s.svc.GetSimpleAccessScope(ctx, &v1.ResourceByID{Id: "someid"})
	s.ErrorIs(err, status.Error(codes.Unimplemented, "feature not enabled"))

	_, err = s.svc.ListSimpleAccessScopes(ctx, &v1.Empty{})
	s.ErrorIs(err, status.Error(codes.Unimplemented, "feature not enabled"))

	_, err = s.svc.PostSimpleAccessScope(ctx, &storage.SimpleAccessScope{})
	s.ErrorIs(err, status.Error(codes.Unimplemented, "feature not enabled"))

	_, err = s.svc.PutSimpleAccessScope(ctx, &storage.SimpleAccessScope{})
	s.ErrorIs(err, status.Error(codes.Unimplemented, "feature not enabled"))

	_, err = s.svc.DeleteSimpleAccessScope(ctx, &v1.ResourceByID{Id: "someid"})
	s.ErrorIs(err, status.Error(codes.Unimplemented, "feature not enabled"))
}
