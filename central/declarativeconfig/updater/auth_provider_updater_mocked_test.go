package updater

import (
	"context"
	"fmt"
	"testing"

	healthMocks "github.com/stackrox/rox/central/declarativeconfig/health/datastore/mocks"
	groupMocks "github.com/stackrox/rox/central/group/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	authProviderMocks "github.com/stackrox/rox/pkg/auth/authproviders/mocks"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestAuthProviderUpdaterMocked(t *testing.T) {
	suite.Run(t, new(mockedAuthProviderUpdaterTestSuite))
}

type mockedAuthProviderUpdaterTestSuite struct {
	suite.Suite

	ctx context.Context

	mockCtrl          *gomock.Controller
	providerDataStore *authProviderMocks.MockStore
	providerRegistry  *authProviderMocks.MockRegistry
	groupDataStore    *groupMocks.MockDataStore
	healthDataStore   *healthMocks.MockDataStore
	updater           ResourceUpdater
}

func (s *mockedAuthProviderUpdaterTestSuite) SetupTest() {
	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Access, resources.Integration),
		),
	)
	s.ctx = declarativeconfig.WithModifyDeclarativeOrImperative(s.ctx)

	s.mockCtrl = gomock.NewController(s.T())

	s.providerDataStore = authProviderMocks.NewMockStore(s.mockCtrl)
	s.providerRegistry = authProviderMocks.NewMockRegistry(s.mockCtrl)
	s.groupDataStore = groupMocks.NewMockDataStore(s.mockCtrl)
	s.healthDataStore = healthMocks.NewMockDataStore(s.mockCtrl)

	s.updater = newAuthProviderUpdater(s.providerDataStore, s.providerRegistry, s.groupDataStore, s.healthDataStore)
}

func (s *mockedAuthProviderUpdaterTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func getAuthProvider(suffix int) *storage.AuthProvider {
	return &storage.AuthProvider{
		Id:         uuid.NewTestUUID(suffix).String(),
		Name:       fmt.Sprintf("test auth provider %d", suffix),
		Type:       "test",
		UiEndpoint: "https://localhost",
	}
}

func (s *mockedAuthProviderUpdaterTestSuite) TestUpsertSuccess() {
	msg := getAuthProvider(1)

	gomock.InOrder(
		s.providerRegistry.EXPECT().DeleteProvider(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil),
		s.providerRegistry.EXPECT().CreateProvider(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil),
	)

	err := s.updater.Upsert(s.ctx, msg)
	s.NoError(err)
}

func (s *mockedAuthProviderUpdaterTestSuite) TestUpsertBadMessage() {
	msg := &storage.SimpleAccessScope{}

	err := s.updater.Upsert(s.ctx, msg)
	s.ErrorIs(err, errox.InvariantViolation)
}

func (s *mockedAuthProviderUpdaterTestSuite) TestUpsertRegistryDeletionFailed() {
	msg := getAuthProvider(1)

	s.providerRegistry.EXPECT().
		DeleteProvider(
			gomock.Any(),
			gomock.Any(),
			gomock.Any(),
			gomock.Any(),
		).
		Return(sac.ErrResourceAccessDenied)

	err := s.updater.Upsert(s.ctx, msg)
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
}

func (s *mockedAuthProviderUpdaterTestSuite) TestUpsertRegistryAdditionFailed() {
	msg := getAuthProvider(1)

	gomock.InOrder(
		s.providerRegistry.EXPECT().DeleteProvider(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil),
		s.providerRegistry.EXPECT().CreateProvider(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errox.AlreadyExists),
	)

	err := s.updater.Upsert(s.ctx, msg)
	s.ErrorIs(err, errox.AlreadyExists)
}
