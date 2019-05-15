package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	storeMocks "github.com/stackrox/rox/central/apitoken/datastore/internal/store/mocks"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

func TestTokenDataStore(t *testing.T) {
	t.Parallel()
	if !features.ScopedAccessControl.Enabled() {
		t.Skip()
	}
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

func (s *apiTokenDataStoreTestSuite) TestEnforcesGet() {
	s.storage.EXPECT().GetTokenOrNil(gomock.Any()).Times(0)

	apitoken, err := s.dataStore.GetTokenOrNil(s.hasNoneCtx, "apitoken")
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(apitoken, "expected return value to be nil")
}

func (s *apiTokenDataStoreTestSuite) TestAllowsGet() {
	s.storage.EXPECT().GetTokenOrNil(gomock.Any()).Return(nil, nil)

	_, err := s.dataStore.GetTokenOrNil(s.hasReadCtx, "apitoken")
	s.NoError(err, "expected no error trying to read with permissions")

	s.storage.EXPECT().GetTokenOrNil(gomock.Any()).Return(nil, nil)

	_, err = s.dataStore.GetTokenOrNil(s.hasWriteCtx, "apitoken")
	s.NoError(err, "expected no error trying to read with permissions")
}

func (s *apiTokenDataStoreTestSuite) TestEnforcesGetMany() {
	s.storage.EXPECT().GetTokens(gomock.Any()).Times(0)

	apitokens, err := s.dataStore.GetTokens(s.hasNoneCtx, &v1.GetAPITokensRequest{})
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(apitokens, "expected return value to be nil")
}

func (s *apiTokenDataStoreTestSuite) TestAllowsGetMany() {
	s.storage.EXPECT().GetTokens(gomock.Any()).Return(nil, nil)

	_, err := s.dataStore.GetTokens(s.hasReadCtx, &v1.GetAPITokensRequest{})
	s.NoError(err, "expected no error trying to read with permissions")

	s.storage.EXPECT().GetTokens(gomock.Any()).Return(nil, nil)

	_, err = s.dataStore.GetTokens(s.hasWriteCtx, &v1.GetAPITokensRequest{})
	s.NoError(err, "expected no error trying to read with permissions")
}

func (s *apiTokenDataStoreTestSuite) TestEnforcesAdd() {
	s.storage.EXPECT().AddToken(gomock.Any()).Times(0)

	err := s.dataStore.AddToken(s.hasNoneCtx, &storage.TokenMetadata{})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.AddToken(s.hasReadCtx, &storage.TokenMetadata{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *apiTokenDataStoreTestSuite) TestAllowsAdd() {
	s.storage.EXPECT().AddToken(gomock.Any()).Return(nil)

	err := s.dataStore.AddToken(s.hasWriteCtx, &storage.TokenMetadata{})
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *apiTokenDataStoreTestSuite) TestEnforcesRevoke() {
	s.storage.EXPECT().RevokeToken(gomock.Any()).Times(0)

	exists, err := s.dataStore.RevokeToken(s.hasNoneCtx, "apitoken")
	s.Error(err, "expected an error trying to write without permissions")
	s.False(exists, "should return false when unable to reach storage.")

	exists, err = s.dataStore.RevokeToken(s.hasReadCtx, "apitoken")
	s.Error(err, "expected an error trying to write without permissions")
	s.False(exists, "should return false when unable to reach storage.")
}

func (s *apiTokenDataStoreTestSuite) TestAllowsRevoke() {
	s.storage.EXPECT().RevokeToken(gomock.Any()).Return(true, nil)

	exists, err := s.dataStore.RevokeToken(s.hasWriteCtx, "apitoken")
	s.NoError(err, "expected no error trying to write with permissions")
	s.True(exists, "should return true when able to reach storage.")
}
