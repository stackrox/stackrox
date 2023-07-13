package datastore

import (
	"context"
	"testing"

	storeMocks "github.com/stackrox/rox/central/authprovider/datastore/internal/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
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
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Access)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Access)))

	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)

	s.dataStore = New(s.storage)
}

func (s *authProviderDataStoreEnforceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *authProviderDataStoreEnforceTestSuite) TestEnforcesGetAll() {
	s.storage.EXPECT().GetAll(gomock.Any()).Return(nil, nil).AnyTimes()

	_, err := s.dataStore.GetAllAuthProviders(s.hasNoneCtx)
	s.ErrorIs(err, sac.ErrResourceAccessDenied)

	_, err = s.dataStore.GetAllAuthProviders(s.hasReadCtx)
	s.NoError(err)

	_, err = s.dataStore.GetAllAuthProviders(s.hasWriteCtx)
	s.NoError(err)
}

func (s *authProviderDataStoreEnforceTestSuite) TestEnforcesAdd() {
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Times(0)

	err := s.dataStore.AddAuthProvider(s.hasNoneCtx, &storage.AuthProvider{})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.AddAuthProvider(s.hasReadCtx, &storage.AuthProvider{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *authProviderDataStoreEnforceTestSuite) TestEnforcesUpdate() {
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Times(0)

	err := s.dataStore.UpdateAuthProvider(s.hasNoneCtx, &storage.AuthProvider{})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.UpdateAuthProvider(s.hasReadCtx, &storage.AuthProvider{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *authProviderDataStoreEnforceTestSuite) TestEnforcesRemove() {
	s.storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)

	err := s.dataStore.RemoveAuthProvider(s.hasNoneCtx, "id", false)
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.RemoveAuthProvider(s.hasReadCtx, "id", false)
	s.Error(err, "expected an error trying to write without permissions")
}

// Test for things that should be allowed by SAC and to confirm storage is used correctly.
func TestAuthProviderDataStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(authProviderDataStoreTestSuite))
}

type authProviderDataStoreTestSuite struct {
	suite.Suite

	hasNoneCtx             context.Context
	hasReadCtx             context.Context
	hasWriteCtx            context.Context
	hasWriteDeclarativeCtx context.Context

	storage   *storeMocks.MockStore
	dataStore authproviders.Store

	mockCtrl *gomock.Controller
}

func (s *authProviderDataStoreTestSuite) SetupTest() {
	s.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Access)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Access)))
	s.hasWriteDeclarativeCtx = declarativeconfig.WithModifyDeclarativeResource(s.hasWriteCtx)

	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)

	s.dataStore = New(s.storage)
}

func (s *authProviderDataStoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *authProviderDataStoreTestSuite) TestAllowsAdd() {
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.storage.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, nil).Times(1)

	err := s.dataStore.AddAuthProvider(s.hasWriteCtx, &storage.AuthProvider{})
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *authProviderDataStoreTestSuite) TestErrorOnAdd() {
	s.storage.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(true, nil)

	err := s.dataStore.AddAuthProvider(s.hasWriteCtx, &storage.AuthProvider{})
	s.Error(err)
}

func (s *authProviderDataStoreTestSuite) TestGetFiltered() {
	authProviders := []*storage.AuthProvider{
		{
			Id:   "some-id-1",
			Name: "some-name-1",
		},
		{
			Id:   "some-id-2",
			Name: "some-name-2",
		},
	}
	s.storage.EXPECT().GetAll(gomock.Any()).Return(authProviders, nil)

	filteredAuthProviders, err := s.dataStore.GetAuthProvidersFiltered(s.hasReadCtx, func(authProvider *storage.AuthProvider) bool {
		return authProvider.GetName() == "some-name-1"
	})
	s.NoError(err)
	s.Len(filteredAuthProviders, 1)
	s.ElementsMatch(filteredAuthProviders, []*storage.AuthProvider{authProviders[0]})
}

func (s *authProviderDataStoreTestSuite) TestAllowsUpdate() {
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&storage.AuthProvider{}, true, nil).Times(1)

	err := s.dataStore.UpdateAuthProvider(s.hasWriteCtx, &storage.AuthProvider{})
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *authProviderDataStoreTestSuite) TestErrorOnUpdate() {
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, false, nil).Times(1)

	err := s.dataStore.UpdateAuthProvider(s.hasWriteCtx, &storage.AuthProvider{})
	s.Error(err)
}

func (s *authProviderDataStoreTestSuite) TestAllowsRemove() {
	s.storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&storage.AuthProvider{}, true, nil).Times(1)

	err := s.dataStore.RemoveAuthProvider(s.hasWriteCtx, "id", false)
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *authProviderDataStoreTestSuite) TestUpdateMutableToImmutable() {
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&storage.AuthProvider{
		Id:   "id",
		Name: "name",
		Traits: &storage.Traits{
			MutabilityMode: storage.Traits_ALLOW_MUTATE,
		},
	}, true, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := s.dataStore.UpdateAuthProvider(s.hasWriteCtx, &storage.AuthProvider{})
	s.NoError(err)
}

func (s *authProviderDataStoreTestSuite) TestUpdateImmutableNoForce() {
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&storage.AuthProvider{
		Id:   "id",
		Name: "name",
		Traits: &storage.Traits{
			MutabilityMode: storage.Traits_ALLOW_MUTATE_FORCED,
		},
	}, true, nil).Times(1)

	err := s.dataStore.UpdateAuthProvider(s.hasWriteCtx, &storage.AuthProvider{})
	s.ErrorIs(err, errox.InvalidArgs)
}

func (s *authProviderDataStoreTestSuite) TestDeleteImmutableNoForce() {
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&storage.AuthProvider{
		Id:   "id",
		Name: "name",
		Traits: &storage.Traits{
			MutabilityMode: storage.Traits_ALLOW_MUTATE_FORCED,
		},
	}, true, nil).Times(1)

	err := s.dataStore.RemoveAuthProvider(s.hasWriteCtx, "id", false)
	s.ErrorIs(err, errox.InvalidArgs)
}

func (s *authProviderDataStoreTestSuite) TestDeleteImmutableForce() {
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&storage.AuthProvider{
		Id:   "id",
		Name: "name",
		Traits: &storage.Traits{
			MutabilityMode: storage.Traits_ALLOW_MUTATE,
		},
	}, true, nil).Times(1)
	s.storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := s.dataStore.RemoveAuthProvider(s.hasWriteCtx, "id", true)
	s.NoError(err)
}

func (s *authProviderDataStoreTestSuite) TestDeleteDeclarativeViaAPI() {
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&storage.AuthProvider{
		Id:   "id",
		Name: "name",
		Traits: &storage.Traits{
			Origin: storage.Traits_DECLARATIVE,
		},
	}, true, nil).Times(1)

	err := s.dataStore.RemoveAuthProvider(s.hasWriteCtx, "id", false)
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *authProviderDataStoreTestSuite) TestDeleteDeclarativeSuccess() {
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&storage.AuthProvider{
		Id:   "id",
		Name: "name",
		Traits: &storage.Traits{
			Origin: storage.Traits_DECLARATIVE,
		},
	}, true, nil).Times(1)
	s.storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := s.dataStore.RemoveAuthProvider(s.hasWriteDeclarativeCtx, "id", false)
	s.NoError(err)
}

func (s *authProviderDataStoreTestSuite) TestUpdateDeclarativeViaAPI() {
	ap := &storage.AuthProvider{
		Id:   "id",
		Name: "name",
		Traits: &storage.Traits{
			Origin: storage.Traits_DECLARATIVE,
		},
	}
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(ap, true, nil).Times(1)

	err := s.dataStore.UpdateAuthProvider(s.hasWriteCtx, ap)
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *authProviderDataStoreTestSuite) TestUpdateDeclarativeSuccess() {
	ap := &storage.AuthProvider{
		Id:   "id",
		Name: "name",
		Traits: &storage.Traits{
			Origin: storage.Traits_DECLARATIVE,
		},
	}
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(ap, true, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := s.dataStore.UpdateAuthProvider(s.hasWriteDeclarativeCtx, ap)
	s.NoError(err)
}

func (s *authProviderDataStoreTestSuite) TestDeleteImperativeDeclaratively() {
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&storage.AuthProvider{
		Id:   "id",
		Name: "name",
		Traits: &storage.Traits{
			Origin: storage.Traits_IMPERATIVE,
		},
	}, true, nil).Times(1)

	err := s.dataStore.RemoveAuthProvider(s.hasWriteDeclarativeCtx, "id", false)
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *authProviderDataStoreTestSuite) TestUpdateImperativeDeclaratively() {
	ap := &storage.AuthProvider{
		Id:   "id",
		Name: "name",
		Traits: &storage.Traits{
			Origin: storage.Traits_IMPERATIVE,
		},
	}
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(ap, true, nil).Times(1)

	err := s.dataStore.UpdateAuthProvider(s.hasWriteDeclarativeCtx, ap)
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *authProviderDataStoreTestSuite) TestAddDeclarativeViaAPI() {
	ap := &storage.AuthProvider{
		Id:   "id",
		Name: "name",
		Traits: &storage.Traits{
			Origin: storage.Traits_DECLARATIVE,
		},
	}

	err := s.dataStore.AddAuthProvider(s.hasWriteCtx, ap)
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *authProviderDataStoreTestSuite) TestAddDeclarativeSuccess() {
	ap := &storage.AuthProvider{
		Id:   "id",
		Name: "name",
		Traits: &storage.Traits{
			Origin: storage.Traits_DECLARATIVE,
		},
	}
	s.storage.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := s.dataStore.AddAuthProvider(s.hasWriteDeclarativeCtx, ap)
	s.NoError(err)
}

func (s *authProviderDataStoreTestSuite) TestAddImperativeDeclaratively() {
	ap := &storage.AuthProvider{
		Id:   "id",
		Name: "name",
		Traits: &storage.Traits{
			Origin: storage.Traits_IMPERATIVE,
		},
	}

	err := s.dataStore.AddAuthProvider(s.hasWriteDeclarativeCtx, ap)
	s.ErrorIs(err, errox.NotAuthorized)
}
