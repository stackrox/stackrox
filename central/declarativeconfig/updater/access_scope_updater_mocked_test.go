package updater

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	healthMocks "github.com/stackrox/rox/central/declarativeconfig/health/datastore/mocks"
	roleMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protomock"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestAccessScopeUpdaterMocked(t *testing.T) {
	suite.Run(t, new(mockedAccessScopeUpdaterTestSuite))
}

func getAccessScope(suffix int) *storage.SimpleAccessScope {
	return &storage.SimpleAccessScope{
		Id:          uuid.NewTestUUID(suffix).String(),
		Name:        fmt.Sprintf("test-%d", suffix),
		Description: "",
		Rules:       &storage.SimpleAccessScope_Rules{IncludedClusters: []string{"cluster1"}},
		Traits:      &storage.Traits{Origin: storage.Traits_DECLARATIVE},
	}
}

type mockedAccessScopeUpdaterTestSuite struct {
	suite.Suite

	ctx context.Context

	mockCtrl *gomock.Controller
	roleDS   *roleMocks.MockDataStore
	healthDS *healthMocks.MockDataStore
	updater  ResourceUpdater
}

func (s *mockedAccessScopeUpdaterTestSuite) SetupTest() {
	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Access, resources.Integration),
		),
	)
	s.ctx = declarativeconfig.WithModifyDeclarativeOrImperative(s.ctx)

	s.mockCtrl = gomock.NewController(s.T())
	s.roleDS = roleMocks.NewMockDataStore(s.mockCtrl)
	s.healthDS = healthMocks.NewMockDataStore(s.mockCtrl)
	s.updater = newAccessScopeUpdater(s.roleDS, s.healthDS)
}

func (s *mockedAccessScopeUpdaterTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *mockedAccessScopeUpdaterTestSuite) TestUpsertBadMessage() {
	msg := &storage.PermissionSet{
		Id:   uuid.NewTestUUID(1).String(),
		Name: "Test permission set",
	}

	err := s.updater.Upsert(s.ctx, msg)
	s.ErrorIs(err, errox.InvariantViolation)
}

func (s *mockedAccessScopeUpdaterTestSuite) TestUpsertSuccess() {
	msg := getAccessScope(1)

	s.roleDS.EXPECT().
		UpsertAccessScope(
			gomock.Any(),
			protomock.GoMockMatcherEqualMessage(msg),
		).
		Times(1).
		Return(nil)

	err := s.updater.Upsert(s.ctx, msg)
	s.NoError(err)
}

func (s *mockedAccessScopeUpdaterTestSuite) TestUpsertTimeout() {
	msg := getAccessScope(1)
	expectedErr := errors.New("timeout")

	s.roleDS.EXPECT().
		UpsertAccessScope(
			gomock.Any(),
			protomock.GoMockMatcherEqualMessage(msg),
		).
		Times(1).
		Return(expectedErr)

	err := s.updater.Upsert(s.ctx, msg)
	s.ErrorIs(err, expectedErr)
}

func (s *mockedAccessScopeUpdaterTestSuite) TestDeleteResourcesGetFilteredError() {
	expectedErr := errox.NotFound
	s.roleDS.EXPECT().
		GetAccessScopesFiltered(gomock.Any(), gomock.Any()).
		Times(1).
		Return(nil, expectedErr)

	deleteFailedIDs, err := s.updater.DeleteResources(s.ctx)
	s.ErrorIs(err, expectedErr)
	s.Empty(deleteFailedIDs)
}

func (s *mockedAccessScopeUpdaterTestSuite) TestDeleteResourcesAllRemovedNoError() {
	scopes := []*storage.SimpleAccessScope{
		getAccessScope(1),
		getAccessScope(2),
		getAccessScope(3),
	}

	gomock.InOrder(
		s.roleDS.EXPECT().GetAccessScopesFiltered(gomock.Any(), gomock.Any()).Return(scopes, nil),
		s.roleDS.EXPECT().RemoveAccessScope(gomock.Any(), scopes[0].GetId()).Return(nil),
		s.roleDS.EXPECT().RemoveAccessScope(gomock.Any(), scopes[1].GetId()).Return(nil),
		s.roleDS.EXPECT().RemoveAccessScope(gomock.Any(), scopes[2].GetId()).Return(nil),
	)

	failedDeleteIDs, err := s.updater.DeleteResources(s.ctx)
	s.NoError(err)
	s.Empty(failedDeleteIDs)
}

func (s *mockedAccessScopeUpdaterTestSuite) TestDeleteResourcesSomeRemovedNoError() {
	scopes := []*storage.SimpleAccessScope{
		getAccessScope(1),
		getAccessScope(2),
		getAccessScope(3),
	}

	gomock.InOrder(
		s.roleDS.EXPECT().GetAccessScopesFiltered(gomock.Any(), gomock.Any()).Return([]*storage.SimpleAccessScope{scopes[0], scopes[2]}, nil),
		s.roleDS.EXPECT().RemoveAccessScope(gomock.Any(), scopes[0].GetId()).Return(nil),
		s.roleDS.EXPECT().RemoveAccessScope(gomock.Any(), scopes[2].GetId()).Return(nil),
	)

	failedDeleteIDs, err := s.updater.DeleteResources(s.ctx, scopes[1].GetId())
	s.NoError(err)
	s.Empty(failedDeleteIDs)
}

func (s *mockedAccessScopeUpdaterTestSuite) TestDeleteResourcesSomeRemovedWithError() {
	scopes := []*storage.SimpleAccessScope{
		getAccessScope(1),
		getAccessScope(2),
		getAccessScope(3),
	}
	removeErr := errox.NotFound

	gomock.InOrder(
		s.roleDS.EXPECT().GetAccessScopesFiltered(gomock.Any(), gomock.Any()).Return([]*storage.SimpleAccessScope{scopes[0], scopes[2]}, nil),
		s.roleDS.EXPECT().RemoveAccessScope(gomock.Any(), scopes[0].GetId()).Return(nil),
		s.roleDS.EXPECT().RemoveAccessScope(gomock.Any(), scopes[2].GetId()).Return(removeErr),
		s.healthDS.EXPECT().UpdateStatusForDeclarativeConfig(gomock.Any(), scopes[2].GetId(), removeErr).Return(nil),
	)

	failedDeleteIDs, err := s.updater.DeleteResources(s.ctx, scopes[1].GetId())
	s.Error(err)
	s.ElementsMatch([]string{scopes[2].GetId()}, failedDeleteIDs)
}

func (s *mockedAccessScopeUpdaterTestSuite) TestDeleteResourcesSomeOrphaned() {
	scopes := []*storage.SimpleAccessScope{
		getAccessScope(1),
		getAccessScope(2),
		getAccessScope(3),
	}
	removeErr := errox.ReferencedByAnotherObject
	updatedScope := scopes[2].CloneVT()
	updatedScope.Traits = &storage.Traits{Origin: storage.Traits_DECLARATIVE_ORPHANED}

	gomock.InOrder(
		s.roleDS.EXPECT().GetAccessScopesFiltered(gomock.Any(), gomock.Any()).Return([]*storage.SimpleAccessScope{scopes[0], scopes[2]}, nil),
		s.roleDS.EXPECT().RemoveAccessScope(gomock.Any(), scopes[0].GetId()).Return(nil),
		s.roleDS.EXPECT().RemoveAccessScope(gomock.Any(), scopes[2].GetId()).Return(removeErr),
		s.healthDS.EXPECT().UpdateStatusForDeclarativeConfig(gomock.Any(), scopes[2].GetId(), removeErr).Return(nil),
		s.roleDS.EXPECT().UpsertAccessScope(gomock.Any(), protomock.GoMockMatcherEqualMessage(updatedScope)).Return(nil),
	)

	failedDeleteIDs, err := s.updater.DeleteResources(s.ctx, scopes[1].GetId())
	s.Error(err)
	s.ElementsMatch([]string{scopes[2].GetId()}, failedDeleteIDs)
}
