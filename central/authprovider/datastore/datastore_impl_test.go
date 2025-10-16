package datastore

import (
	"context"
	"testing"

	storeMocks "github.com/stackrox/rox/central/authprovider/datastore/internal/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// Separate tests for testing that things are rejected by SAC.
func TestSACEnforceAuthProviderDataStore(t *testing.T) {
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
			sac.ResourceScopeKeys(resources.Access)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Access)))

	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)

	s.dataStore = New(s.storage)
}

func (s *authProviderDataStoreEnforceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *authProviderDataStoreEnforceTestSuite) TestEnforcesAuthProviderExistsWithName() {
	s.storage.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	const testProviderName = "Test Auth Provider"

	_, err := s.dataStore.AuthProviderExistsWithName(s.hasNoneCtx, testProviderName)
	s.ErrorIs(err, sac.ErrResourceAccessDenied)

	_, err = s.dataStore.AuthProviderExistsWithName(s.hasReadCtx, testProviderName)
	s.NoError(err)

	_, err = s.dataStore.AuthProviderExistsWithName(s.hasWriteCtx, testProviderName)
	s.NoError(err)
}

func (s *authProviderDataStoreEnforceTestSuite) TestEnforcesProcessAuthProviders() {
	err := s.dataStore.ForEachAuthProvider(s.hasNoneCtx, nil)
	s.ErrorIs(err, sac.ErrResourceAccessDenied)

	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).Return(nil).Times(2)

	err = s.dataStore.ForEachAuthProvider(s.hasReadCtx, nil)
	s.NoError(err)

	err = s.dataStore.ForEachAuthProvider(s.hasWriteCtx, nil)
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
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Access)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
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

	ap := &storage.AuthProvider{}
	ap.SetId("test")
	ap.SetName("test")
	ap.SetLoginUrl("test")
	err := s.dataStore.AddAuthProvider(s.hasWriteCtx, ap)
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *authProviderDataStoreTestSuite) TestErrorOnAdd() {
	s.storage.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(true, nil)

	ap := &storage.AuthProvider{}
	ap.SetId("test")
	ap.SetName("test")
	ap.SetLoginUrl("test")
	err := s.dataStore.AddAuthProvider(s.hasWriteCtx, ap)
	s.Error(err)
}

func (s *authProviderDataStoreTestSuite) TestAuthProviderExistsWithName() {
	testProviderName := "Test Auth Provider"
	s.storage.EXPECT().
		Search(gomock.Any(), gomock.Any()).
		Times(1).
		Return([]search.Result{{ID: "1234"}}, nil)
	exists, err := s.dataStore.AuthProviderExistsWithName(s.hasReadCtx, testProviderName)
	s.NoError(err)
	s.True(exists)

	s.storage.EXPECT().
		Search(gomock.Any(), gomock.Any()).
		Times(1).
		Return(nil, nil)
	exists, err = s.dataStore.AuthProviderExistsWithName(s.hasReadCtx, testProviderName)
	s.NoError(err)
	s.False(exists)
}

func (s *authProviderDataStoreTestSuite) TestGetFiltered() {
	ap := &storage.AuthProvider{}
	ap.SetId("some-id-1")
	ap.SetName("some-name-1")
	ap2 := &storage.AuthProvider{}
	ap2.SetId("some-id-2")
	ap2.SetName("some-name-2")
	authProviders := []*storage.AuthProvider{
		ap,
		ap2,
	}
	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, fn func(p *storage.AuthProvider) error) error {
		for _, p := range authProviders {
			if err := fn(p); err != nil {
				return err
			}
		}
		return nil
	})

	filteredAuthProviders, err := s.dataStore.GetAuthProvidersFiltered(s.hasReadCtx, func(authProvider *storage.AuthProvider) bool {
		return authProvider.GetName() == "some-name-1"
	})
	s.NoError(err)
	s.Len(filteredAuthProviders, 1)
	protoassert.ElementsMatch(s.T(), filteredAuthProviders, []*storage.AuthProvider{authProviders[0]})
}

func (s *authProviderDataStoreTestSuite) TestAllowsUpdate() {
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&storage.AuthProvider{}, true, nil).Times(1)

	ap := &storage.AuthProvider{}
	ap.SetId("test")
	ap.SetName("test")
	ap.SetLoginUrl("test")
	err := s.dataStore.UpdateAuthProvider(s.hasWriteCtx, ap)
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *authProviderDataStoreTestSuite) TestErrorOnUpdate() {
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, false, nil).Times(1)

	ap := &storage.AuthProvider{}
	ap.SetId("test")
	ap.SetName("test")
	ap.SetLoginUrl("test")
	err := s.dataStore.UpdateAuthProvider(s.hasWriteCtx, ap)
	s.Error(err)
}

func (s *authProviderDataStoreTestSuite) TestAllowsRemove() {
	s.storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&storage.AuthProvider{}, true, nil).Times(1)

	err := s.dataStore.RemoveAuthProvider(s.hasWriteCtx, "id", false)
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *authProviderDataStoreTestSuite) TestUpdateMutableToImmutable() {
	traits := &storage.Traits{}
	traits.SetMutabilityMode(storage.Traits_ALLOW_MUTATE)
	ap := &storage.AuthProvider{}
	ap.SetId("id")
	ap.SetName("name")
	ap.SetLoginUrl("test")
	ap.SetTraits(traits)
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(ap, true, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	ap2 := &storage.AuthProvider{}
	ap2.SetId("id")
	ap2.SetName("test")
	ap2.SetLoginUrl("test")
	err := s.dataStore.UpdateAuthProvider(s.hasWriteCtx, ap2)
	s.NoError(err)
}

func (s *authProviderDataStoreTestSuite) TestUpdateImmutableNoForce() {
	traits := &storage.Traits{}
	traits.SetMutabilityMode(storage.Traits_ALLOW_MUTATE_FORCED)
	ap := &storage.AuthProvider{}
	ap.SetId("id")
	ap.SetName("name")
	ap.SetLoginUrl("test")
	ap.SetTraits(traits)
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(ap, true, nil).Times(1)

	ap2 := &storage.AuthProvider{}
	ap2.SetId("id")
	ap2.SetName("test")
	ap2.SetLoginUrl("test")
	err := s.dataStore.UpdateAuthProvider(s.hasWriteCtx, ap2)
	s.ErrorIs(err, errox.InvalidArgs)
}

func (s *authProviderDataStoreTestSuite) TestDeleteImmutableNoForce() {
	traits := &storage.Traits{}
	traits.SetMutabilityMode(storage.Traits_ALLOW_MUTATE_FORCED)
	ap := &storage.AuthProvider{}
	ap.SetId("id")
	ap.SetName("name")
	ap.SetLoginUrl("test")
	ap.SetTraits(traits)
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(ap, true, nil).Times(1)

	err := s.dataStore.RemoveAuthProvider(s.hasWriteCtx, "id", false)
	s.ErrorIs(err, errox.InvalidArgs)
}

func (s *authProviderDataStoreTestSuite) TestDeleteImmutableForce() {
	traits := &storage.Traits{}
	traits.SetMutabilityMode(storage.Traits_ALLOW_MUTATE)
	ap := &storage.AuthProvider{}
	ap.SetId("id")
	ap.SetName("name")
	ap.SetLoginUrl("test")
	ap.SetTraits(traits)
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(ap, true, nil).Times(1)
	s.storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := s.dataStore.RemoveAuthProvider(s.hasWriteCtx, "id", true)
	s.NoError(err)
}

func (s *authProviderDataStoreTestSuite) TestDeleteDeclarativeViaAPI() {
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_DECLARATIVE)
	ap := &storage.AuthProvider{}
	ap.SetId("id")
	ap.SetName("name")
	ap.SetLoginUrl("test")
	ap.SetTraits(traits)
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(ap, true, nil).Times(1)

	err := s.dataStore.RemoveAuthProvider(s.hasWriteCtx, "id", false)
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *authProviderDataStoreTestSuite) TestDeleteDeclarativeSuccess() {
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_DECLARATIVE)
	ap := &storage.AuthProvider{}
	ap.SetId("id")
	ap.SetName("name")
	ap.SetLoginUrl("test")
	ap.SetTraits(traits)
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(ap, true, nil).Times(1)
	s.storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := s.dataStore.RemoveAuthProvider(s.hasWriteDeclarativeCtx, "id", false)
	s.NoError(err)
}

func (s *authProviderDataStoreTestSuite) TestUpdateDeclarativeViaAPI() {
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_DECLARATIVE)
	ap := &storage.AuthProvider{}
	ap.SetId("id")
	ap.SetName("name")
	ap.SetLoginUrl("test")
	ap.SetTraits(traits)
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(ap, true, nil).Times(1)

	err := s.dataStore.UpdateAuthProvider(s.hasWriteCtx, ap)
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *authProviderDataStoreTestSuite) TestUpdateDeclarativeSuccess() {
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_DECLARATIVE)
	ap := &storage.AuthProvider{}
	ap.SetId("id")
	ap.SetName("name")
	ap.SetLoginUrl("test")
	ap.SetTraits(traits)
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(ap, true, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := s.dataStore.UpdateAuthProvider(s.hasWriteDeclarativeCtx, ap)
	s.NoError(err)
}

func (s *authProviderDataStoreTestSuite) TestDeleteImperativeDeclaratively() {
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_IMPERATIVE)
	ap := &storage.AuthProvider{}
	ap.SetId("id")
	ap.SetName("name")
	ap.SetLoginUrl("test")
	ap.SetTraits(traits)
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(ap, true, nil).Times(1)

	err := s.dataStore.RemoveAuthProvider(s.hasWriteDeclarativeCtx, "id", false)
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *authProviderDataStoreTestSuite) TestUpdateImperativeDeclaratively() {
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_IMPERATIVE)
	ap := &storage.AuthProvider{}
	ap.SetId("id")
	ap.SetName("name")
	ap.SetLoginUrl("test")
	ap.SetTraits(traits)
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(ap, true, nil).Times(1)

	err := s.dataStore.UpdateAuthProvider(s.hasWriteDeclarativeCtx, ap)
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *authProviderDataStoreTestSuite) TestAddDeclarativeViaAPI() {
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_DECLARATIVE)
	ap := &storage.AuthProvider{}
	ap.SetId("id")
	ap.SetName("name")
	ap.SetLoginUrl("test")
	ap.SetTraits(traits)

	err := s.dataStore.AddAuthProvider(s.hasWriteCtx, ap)
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *authProviderDataStoreTestSuite) TestAddDeclarativeSuccess() {
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_DECLARATIVE)
	ap := &storage.AuthProvider{}
	ap.SetId("id")
	ap.SetName("name")
	ap.SetLoginUrl("test")
	ap.SetTraits(traits)
	s.storage.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := s.dataStore.AddAuthProvider(s.hasWriteDeclarativeCtx, ap)
	s.NoError(err)
}

func (s *authProviderDataStoreTestSuite) TestAddImperativeDeclaratively() {
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_IMPERATIVE)
	ap := &storage.AuthProvider{}
	ap.SetId("id")
	ap.SetName("name")
	ap.SetLoginUrl("test")
	ap.SetTraits(traits)

	err := s.dataStore.AddAuthProvider(s.hasWriteDeclarativeCtx, ap)
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *authProviderDataStoreTestSuite) TestValidateAuthProvider() {
	cases := map[string]struct {
		ap  *storage.AuthProvider
		err error
	}{
		"empty auth provider should return error": {
			ap:  &storage.AuthProvider{},
			err: errox.InvalidArgs,
		},
		"empty ID should return an error": {
			ap: storage.AuthProvider_builder{
				Name:     "test",
				LoginUrl: "test",
			}.Build(),
			err: errox.InvalidArgs,
		},
		"empty name should return an error": {
			ap: storage.AuthProvider_builder{
				Id:       "test-id",
				LoginUrl: "test",
			}.Build(),
			err: errox.InvalidArgs,
		},
		"empty login URL should return an error": {
			ap: storage.AuthProvider_builder{
				Id:   "test-id",
				Name: "test",
			}.Build(),
			err: errox.InvalidArgs,
		},
		"all required fields set should not return an error": {
			ap: storage.AuthProvider_builder{
				Id:       "test-id",
				Name:     "test",
				LoginUrl: "test",
			}.Build(),
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			s.ErrorIs(validateAuthProvider(tc.ap), tc.err)
		})
	}
}
