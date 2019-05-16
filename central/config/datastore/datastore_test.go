package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	storeMocks "github.com/stackrox/rox/central/config/store/mocks"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

func TestGroupDataStore(t *testing.T) {
	t.Parallel()
	if !features.ScopedAccessControl.Enabled() {
		t.Skip()
	}
	suite.Run(t, new(groupDataStoreTestSuite))
}

type groupDataStoreTestSuite struct {
	suite.Suite

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	dataStore DataStore
	storage   *storeMocks.MockStore

	mockCtrl *gomock.Controller
}

func (s *groupDataStoreTestSuite) SetupTest() {
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

func (s *groupDataStoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *groupDataStoreTestSuite) TestEnforcesGet() {
	s.storage.EXPECT().GetConfig().Times(0)

	config, err := s.dataStore.GetConfig(s.hasNoneCtx)
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(config, "expected return value to be nil")
}

func (s *groupDataStoreTestSuite) TestAllowsGet() {
	s.storage.EXPECT().GetConfig().Return(nil, nil)

	_, err := s.dataStore.GetConfig(s.hasNoneCtx)
	s.NoError(err, "expected no error trying to read with permissions")

	s.storage.EXPECT().GetConfig().Return(nil, nil)

	_, err = s.dataStore.GetConfig(s.hasNoneCtx)
	s.NoError(err, "expected no error trying to read with permissions")
}

func (s *groupDataStoreTestSuite) TestEnforcesUpdate() {
	s.storage.EXPECT().UpdateConfig(gomock.Any()).Times(0)

	err := s.dataStore.UpdateConfig(s.hasNoneCtx, &storage.Config{})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.UpdateConfig(s.hasReadCtx, &storage.Config{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *groupDataStoreTestSuite) TestAllowsUpdate() {
	s.storage.EXPECT().UpdateConfig(gomock.Any()).Return(nil)

	err := s.dataStore.UpdateConfig(s.hasWriteCtx, &storage.Config{})
	s.NoError(err, "expected no error trying to write with permissions")
}
