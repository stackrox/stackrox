package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	storeMocks "github.com/stackrox/rox/central/group/datastore/internal/store/mocks"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

func TestGroupDataStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(groupDataStoreTestSuite))
}

type groupDataStoreTestSuite struct {
	suite.Suite

	hasNoneCtx        context.Context
	hasReadCtx        context.Context
	hasWriteCtx       context.Context
	hasWriteAccessCtx context.Context

	dataStore DataStore
	storage   *storeMocks.MockStore

	mockCtrl *gomock.Controller
}

func (s *groupDataStoreTestSuite) SetupTest() {
	s.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Group)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Group)))
	s.hasWriteAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Access)))

	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)
	s.dataStore = New(s.storage)
}

func (s *groupDataStoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *groupDataStoreTestSuite) TestEnforcesGet() {
	s.storage.EXPECT().Get(gomock.Any()).Times(0)

	group, err := s.dataStore.Get(s.hasNoneCtx, &storage.GroupProperties{})
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(group, "expected return value to be nil")
}

func (s *groupDataStoreTestSuite) TestAllowsGet() {
	s.storage.EXPECT().Get(gomock.Any()).Return(nil, nil)

	_, err := s.dataStore.Get(s.hasReadCtx, &storage.GroupProperties{})
	s.NoError(err, "expected no error trying to read with permissions")

	s.storage.EXPECT().Get(gomock.Any()).Return(nil, nil).Times(2)

	_, err = s.dataStore.Get(s.hasWriteCtx, &storage.GroupProperties{})
	s.NoError(err, "expected no error trying to read with permissions")

	_, err = s.dataStore.Get(s.hasWriteAccessCtx, &storage.GroupProperties{})
	s.NoError(err, "expected no error trying to read with Access permissions")
}

func (s *groupDataStoreTestSuite) TestEnforcesGetAll() {
	s.storage.EXPECT().GetAll().Times(0)

	groups, err := s.dataStore.GetAll(s.hasNoneCtx)
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(groups, "expected return value to be nil")
}

func (s *groupDataStoreTestSuite) TestAllowsGetAll() {
	s.storage.EXPECT().GetAll().Return(nil, nil)

	_, err := s.dataStore.GetAll(s.hasReadCtx)
	s.NoError(err, "expected no error trying to read with permissions")

	s.storage.EXPECT().GetAll().Return(nil, nil).Times(2)

	_, err = s.dataStore.GetAll(s.hasWriteCtx)
	s.NoError(err, "expected no error trying to read with permissions")

	_, err = s.dataStore.GetAll(s.hasWriteAccessCtx)
	s.NoError(err, "expected no error trying to read with Access permissions")
}

func (s *groupDataStoreTestSuite) TestEnforcesWalk() {
	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).Times(0)

	groups, err := s.dataStore.Walk(s.hasNoneCtx, "provider", nil)
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(groups, "expected return value to be nil")
}

func (s *groupDataStoreTestSuite) TestAllowsWalk() {
	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).Return(nil, nil)

	_, err := s.dataStore.Walk(s.hasReadCtx, "provider", nil)
	s.NoError(err, "expected no error trying to read with permissions")

	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).Return(nil, nil).Times(2)

	_, err = s.dataStore.Walk(s.hasWriteCtx, "provider", nil)
	s.NoError(err, "expected no error trying to read with permissions")

	_, err = s.dataStore.Walk(s.hasWriteAccessCtx, "provider", nil)
	s.NoError(err, "expected no error trying to read with Access permissions")
}

func (s *groupDataStoreTestSuite) TestEnforcesAdd() {
	s.storage.EXPECT().Add(gomock.Any()).Times(0)

	err := s.dataStore.Add(s.hasNoneCtx, &storage.Group{})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.Add(s.hasReadCtx, &storage.Group{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *groupDataStoreTestSuite) TestAllowsAdd() {
	s.storage.EXPECT().Add(gomock.Any()).Return(nil).Times(2)

	err := s.dataStore.Add(s.hasWriteCtx, &storage.Group{})
	s.NoError(err, "expected no error trying to write with permissions")

	err = s.dataStore.Add(s.hasWriteAccessCtx, &storage.Group{})
	s.NoError(err, "expected no error trying to write with Access permissions")
}

func (s *groupDataStoreTestSuite) TestEnforcesUpdate() {
	s.storage.EXPECT().Update(gomock.Any()).Times(0)

	err := s.dataStore.Update(s.hasNoneCtx, &storage.Group{})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.Update(s.hasReadCtx, &storage.Group{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *groupDataStoreTestSuite) TestAllowsUpdate() {
	s.storage.EXPECT().Update(gomock.Any()).Return(nil).Times(2)

	err := s.dataStore.Update(s.hasWriteCtx, &storage.Group{})
	s.NoError(err, "expected no error trying to write with permissions")

	err = s.dataStore.Update(s.hasWriteAccessCtx, &storage.Group{})
	s.NoError(err, "expected no error trying to write with Access permissions")
}

func (s *groupDataStoreTestSuite) TestEnforcesMutate() {
	s.storage.EXPECT().Mutate(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	err := s.dataStore.Mutate(s.hasNoneCtx, []*storage.Group{}, []*storage.Group{}, []*storage.Group{})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.Mutate(s.hasReadCtx, []*storage.Group{}, []*storage.Group{}, []*storage.Group{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *groupDataStoreTestSuite) TestAllowsMutate() {
	s.storage.EXPECT().Mutate(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(2)

	err := s.dataStore.Mutate(s.hasWriteCtx, []*storage.Group{}, []*storage.Group{}, []*storage.Group{})
	s.NoError(err, "expected no error trying to write with permissions")

	err = s.dataStore.Mutate(s.hasWriteAccessCtx, []*storage.Group{}, []*storage.Group{}, []*storage.Group{})
	s.NoError(err, "expected no error trying to write with Access permissions")
}

func (s *groupDataStoreTestSuite) TestEnforcesRemove() {
	s.storage.EXPECT().Remove(gomock.Any()).Times(0)

	err := s.dataStore.Remove(s.hasNoneCtx, &storage.GroupProperties{})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.Remove(s.hasReadCtx, &storage.GroupProperties{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *groupDataStoreTestSuite) TestAllowsRemove() {
	s.storage.EXPECT().Remove(gomock.Any()).Return(nil).Times(2)

	err := s.dataStore.Remove(s.hasWriteCtx, &storage.GroupProperties{})
	s.NoError(err, "expected no error trying to write with permissions")

	err = s.dataStore.Remove(s.hasWriteAccessCtx, &storage.GroupProperties{})
	s.NoError(err, "expected no error trying to write with Access permissions")
}
