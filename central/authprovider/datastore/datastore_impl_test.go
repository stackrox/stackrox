package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	storeMocks "github.com/stackrox/rox/central/authprovider/datastore/internal/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

// Separate tests for testing that things are rejected by SAC.
func TestSACEnforceAuthProviderDataStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(authProviderDataStoreEnforceTestSuite))
}

type authProviderDataStoreEnforceTestSuite struct {
	suite.Suite

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	storage   *storeMocks.MockStore
	dataStore authproviders.Store

	mockCtrl *gomock.Controller
}

func (s *authProviderDataStoreEnforceTestSuite) SetupTest() {
	s.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.AuthProvider)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.AuthProvider)))

	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)

	s.dataStore = New(s.storage)
}

func (s *authProviderDataStoreEnforceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *authProviderDataStoreEnforceTestSuite) TestEnforcesAdd() {
	s.storage.EXPECT().AddAuthProvider(gomock.Any()).Times(0)

	err := s.dataStore.AddAuthProvider(s.hasNoneCtx, &storage.AuthProvider{})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.AddAuthProvider(s.hasReadCtx, &storage.AuthProvider{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *authProviderDataStoreEnforceTestSuite) TestEnforcesUpdate() {
	s.storage.EXPECT().UpdateAuthProvider(gomock.Any()).Times(0)

	err := s.dataStore.UpdateAuthProvider(s.hasNoneCtx, &storage.AuthProvider{})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.UpdateAuthProvider(s.hasReadCtx, &storage.AuthProvider{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *authProviderDataStoreEnforceTestSuite) TestEnforcesRemove() {
	s.storage.EXPECT().RemoveAuthProvider(gomock.Any()).Times(0)

	err := s.dataStore.RemoveAuthProvider(s.hasNoneCtx, "id")
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.RemoveAuthProvider(s.hasReadCtx, "id")
	s.Error(err, "expected an error trying to write without permissions")
}

// Test for things that should be allowed by SAC and to confirm storage is used correctly.
func TestAuthProviderDataStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(authProviderDataStoreTestSuite))
}

type authProviderDataStoreTestSuite struct {
	suite.Suite

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	storage   *storeMocks.MockStore
	dataStore authproviders.Store

	mockCtrl *gomock.Controller
}

func (s *authProviderDataStoreTestSuite) SetupTest() {
	s.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.AuthProvider)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.AuthProvider)))

	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)

	s.dataStore = New(s.storage)
}

func (s *authProviderDataStoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *authProviderDataStoreTestSuite) TestAllowsAdd() {
	s.storage.EXPECT().AddAuthProvider(gomock.Any()).Return(nil)

	err := s.dataStore.AddAuthProvider(s.hasWriteCtx, &storage.AuthProvider{})
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *authProviderDataStoreTestSuite) TestAllowsUpdate() {
	s.storage.EXPECT().UpdateAuthProvider(gomock.Any()).Return(nil)

	err := s.dataStore.UpdateAuthProvider(s.hasWriteCtx, &storage.AuthProvider{})
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *authProviderDataStoreTestSuite) TestAllowsRemove() {
	s.storage.EXPECT().RemoveAuthProvider(gomock.Any()).Return(nil)

	err := s.dataStore.RemoveAuthProvider(s.hasWriteCtx, "id")
	s.NoError(err, "expected no error trying to write with permissions")
}
