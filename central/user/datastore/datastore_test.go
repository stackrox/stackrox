package datastore

import (
	"context"
	"testing"

	storeMocks "github.com/stackrox/rox/central/user/datastore/internal/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestUserDataStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(userDataStoreTestSuite))
}

type userDataStoreTestSuite struct {
	suite.Suite

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	dataStore DataStore
	storage   *storeMocks.MockStore

	mockCtrl *gomock.Controller
}

func (s *userDataStoreTestSuite) SetupTest() {
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

func (s *userDataStoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *userDataStoreTestSuite) TestEnforcesGet() {
	s.storage.EXPECT().GetUser(gomock.Any()).Times(0)

	user, err := s.dataStore.GetUser(s.hasNoneCtx, "user")
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(user, "expected return value to be nil")
}

func (s *userDataStoreTestSuite) TestAllowsGet() {
	s.storage.EXPECT().GetUser(gomock.Any()).Return(nil, nil)

	_, err := s.dataStore.GetUser(s.hasReadCtx, "user")
	s.NoError(err, "expected no error trying to read with permissions")

	s.storage.EXPECT().GetUser(gomock.Any()).Return(nil, nil)

	_, err = s.dataStore.GetUser(s.hasWriteCtx, "user")
	s.NoError(err, "expected no error trying to read with permissions")
}

func (s *userDataStoreTestSuite) TestEnforcesGetAll() {
	s.storage.EXPECT().GetAllUsers().Times(0)

	users, err := s.dataStore.GetAllUsers(s.hasNoneCtx)
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(users, "expected return value to be nil")
}

func (s *userDataStoreTestSuite) TestAllowsGetAll() {
	s.storage.EXPECT().GetAllUsers().Return(nil, nil)

	_, err := s.dataStore.GetAllUsers(s.hasReadCtx)
	s.NoError(err, "expected no error trying to read with permissions")

	s.storage.EXPECT().GetAllUsers().Return(nil, nil)

	_, err = s.dataStore.GetAllUsers(s.hasWriteCtx)
	s.NoError(err, "expected no error trying to read with permissions")
}

func (s *userDataStoreTestSuite) TestEnforcesUpsert() {
	s.storage.EXPECT().Upsert(gomock.Any()).Times(0)

	err := s.dataStore.Upsert(s.hasNoneCtx, &storage.User{})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.Upsert(s.hasReadCtx, &storage.User{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *userDataStoreTestSuite) TestAllowsUpsert() {
	s.storage.EXPECT().Upsert(gomock.Any()).Return(nil)

	err := s.dataStore.Upsert(s.hasWriteCtx, &storage.User{})
	s.NoError(err, "expected no error trying to write with permissions")
}
