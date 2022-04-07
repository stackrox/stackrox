package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	storeMocks "github.com/stackrox/rox/central/apitoken/datastore/internal/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/sac"
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

	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)
	s.dataStore = New(s.storage)
}

func (s *apiTokenDataStoreTestSuite) TeardownTest() {
	s.mockCtrl.Finish()
}

func (s *apiTokenDataStoreTestSuite) TestAddToken() {
	token := &storage.TokenMetadata{Id: "id"}
	s.storage.EXPECT().Upsert(gomock.Any(), token).Return(nil)

	s.NoError(s.dataStore.AddToken(s.hasWriteCtx, token))
}

func (s *apiTokenDataStoreTestSuite) TestGetTokenOrNil() {
	expectedToken := &storage.TokenMetadata{Id: "id"}
	s.storage.EXPECT().Get(gomock.Any(), "id").Return(nil, false, nil)

	token, err := s.dataStore.GetTokenOrNil(s.hasReadCtx, "id")
	s.NoError(err)
	s.Nil(token)

	s.storage.EXPECT().Get(gomock.Any(), "id").Return(expectedToken, true, nil)

	token, err = s.dataStore.GetTokenOrNil(s.hasReadCtx, "id")
	s.NoError(err)
	s.Equal(expectedToken, token)
}

func (s *apiTokenDataStoreTestSuite) TestRevokeToken() {
	expectedToken := &storage.TokenMetadata{Id: "id"}
	s.storage.EXPECT().Get(gomock.Any(), "id").Return(nil, false, nil)

	exists, err := s.dataStore.RevokeToken(s.hasWriteCtx, "id")
	s.NoError(err)
	s.False(exists)

	s.storage.EXPECT().Get(gomock.Any(), "id").Return(expectedToken, true, nil)
	expectedToken.Revoked = true
	s.storage.EXPECT().Upsert(gomock.Any(), expectedToken).Return(nil)

	exists, err = s.dataStore.RevokeToken(s.hasWriteCtx, "id")
	s.NoError(err)
	s.True(exists)
}
