package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	storeMocks "github.com/stackrox/stackrox/central/apitoken/datastore/internal/store/mocks"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

func TestTokenDataStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(apiTokenDataStoreTestSuite))
}

type apiTokenDataStoreTestSuite struct {
	suite.Suite

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	hasReadIntegrationCtx  context.Context
	hasWriteIntegrationCtx context.Context

	dataStore DataStore
	storage   *storeMocks.MockStore

	mockCtrl *gomock.Controller
}

func (s *apiTokenDataStoreTestSuite) SetupTest() {
	s.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.APIToken)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.APIToken)))

	s.hasReadIntegrationCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Integration)))
	s.hasWriteIntegrationCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Integration)))

	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)
	s.dataStore = New(s.storage)
}

func (s *apiTokenDataStoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *apiTokenDataStoreTestSuite) TestAddToken() {
	token := &storage.TokenMetadata{Id: "id"}
	s.storage.EXPECT().Upsert(gomock.Any(), token).Return(nil).MaxTimes(2)

	s.NoError(s.dataStore.AddToken(s.hasWriteCtx, token))

	token.Id = "id2"

	s.NoError(s.dataStore.AddToken(s.hasWriteIntegrationCtx, token))
}

func (s *apiTokenDataStoreTestSuite) TestGetTokenOrNil() {
	expectedToken := &storage.TokenMetadata{Id: "id"}
	s.storage.EXPECT().Get(gomock.Any(), "id").Return(nil, false, nil).MaxTimes(2)

	token, err := s.dataStore.GetTokenOrNil(s.hasReadCtx, "id")
	s.NoError(err)
	s.Nil(token)

	token, err = s.dataStore.GetTokenOrNil(s.hasReadIntegrationCtx, "id")
	s.NoError(err)
	s.Nil(token)

	s.storage.EXPECT().Get(gomock.Any(), "id").Return(expectedToken, true, nil).MaxTimes(2)

	token, err = s.dataStore.GetTokenOrNil(s.hasReadCtx, "id")
	s.NoError(err)
	s.Equal(expectedToken, token)

	token, err = s.dataStore.GetTokenOrNil(s.hasReadIntegrationCtx, "id")
	s.NoError(err)
	s.Equal(expectedToken, token)
}

func (s *apiTokenDataStoreTestSuite) TestRevokeToken() {
	expectedToken := &storage.TokenMetadata{Id: "id"}
	s.storage.EXPECT().Get(gomock.Any(), "id").Return(nil, false, nil).MaxTimes(2)

	exists, err := s.dataStore.RevokeToken(s.hasWriteCtx, "id")
	s.NoError(err)
	s.False(exists)

	exists, err = s.dataStore.RevokeToken(s.hasWriteIntegrationCtx, "id")
	s.NoError(err)
	s.False(exists)

	s.storage.EXPECT().Get(gomock.Any(), "id").Return(expectedToken, true, nil).MaxTimes(2)
	expectedToken.Revoked = true
	s.storage.EXPECT().Upsert(gomock.Any(), expectedToken).Return(nil).MaxTimes(2)

	exists, err = s.dataStore.RevokeToken(s.hasWriteCtx, "id")
	s.NoError(err)
	s.True(exists)

	exists, err = s.dataStore.RevokeToken(s.hasWriteIntegrationCtx, "id")
	s.NoError(err)
	s.True(exists)
}
