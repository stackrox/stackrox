package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	storeMocks "github.com/stackrox/rox/central/config/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

func TestConfigDataStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(configDataStoreTestSuite))
}

type configDataStoreTestSuite struct {
	suite.Suite

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	dataStore DataStore
	storage   *storeMocks.MockStore

	mockCtrl *gomock.Controller
}

func (s *configDataStoreTestSuite) SetupTest() {
	s.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Config)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Config)))

	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)
	s.dataStore = New(s.storage)
}

func (s *configDataStoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *configDataStoreTestSuite) TestEnforcesGet() {
	s.storage.EXPECT().GetConfig().Times(0)

	config, err := s.dataStore.GetConfig(s.hasNoneCtx)
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(config, "expected return value to be nil")
}

func (s *configDataStoreTestSuite) TestAllowsGet() {
	s.storage.EXPECT().GetConfig().Return(nil, nil)

	_, err := s.dataStore.GetConfig(s.hasReadCtx)
	s.NoError(err, "expected no error trying to read with permissions")

	s.storage.EXPECT().GetConfig().Return(nil, nil)

	_, err = s.dataStore.GetConfig(s.hasWriteCtx)
	s.NoError(err, "expected no error trying to read with permissions")
}

func (s *configDataStoreTestSuite) TestEnforcesUpdate() {
	s.storage.EXPECT().UpsertConfig(gomock.Any()).Times(0)

	err := s.dataStore.UpsertConfig(s.hasNoneCtx, &storage.Config{})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.UpsertConfig(s.hasReadCtx, &storage.Config{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *configDataStoreTestSuite) TestAllowsUpdate() {
	s.storage.EXPECT().UpsertConfig(gomock.Any()).Return(nil)

	err := s.dataStore.UpsertConfig(s.hasWriteCtx, &storage.Config{})
	s.NoError(err, "expected no error trying to write with permissions")
}
