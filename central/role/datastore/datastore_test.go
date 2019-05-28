package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	storeMocks "github.com/stackrox/rox/central/role/datastore/internal/store/mocks"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

func TestRoleDataStore(t *testing.T) {
	t.Parallel()
	if !features.ScopedAccessControl.Enabled() {
		t.Skip()
	}
	suite.Run(t, new(roleDataStoreTestSuite))
}

type roleDataStoreTestSuite struct {
	suite.Suite

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	dataStore DataStore
	storage   *storeMocks.MockStore

	mockCtrl *gomock.Controller
}

func (s *roleDataStoreTestSuite) SetupTest() {
	s.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Role)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Role)))

	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)
	s.dataStore = New(s.storage)
}

func (s *roleDataStoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *roleDataStoreTestSuite) TestEnforcesGet() {
	s.storage.EXPECT().GetRole(gomock.Any()).Times(0)

	role, err := s.dataStore.GetRole(s.hasNoneCtx, "someID")
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(role, "expected return value to be nil")
}

func (s *roleDataStoreTestSuite) TestAllowsGet() {
	s.storage.EXPECT().GetRole(gomock.Any()).Return(nil, nil)

	_, err := s.dataStore.GetRole(s.hasReadCtx, "someID")
	s.NoError(err, "expected no error trying to read with permissions")

	s.storage.EXPECT().GetRole(gomock.Any()).Return(nil, nil)

	_, err = s.dataStore.GetRole(s.hasWriteCtx, "someID")
	s.NoError(err, "expected no error trying to read with permissions")
}

func (s *roleDataStoreTestSuite) TestEnforcesGetAll() {
	s.storage.EXPECT().GetAllRoles().Times(0)

	roles, err := s.dataStore.GetAllRoles(s.hasNoneCtx)
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(roles, "expected return value to be nil")
}

func (s *roleDataStoreTestSuite) TestAllowsGetAll() {
	s.storage.EXPECT().GetAllRoles().Return(nil, nil)

	_, err := s.dataStore.GetAllRoles(s.hasReadCtx)
	s.NoError(err, "expected no error trying to read with permissions")

	s.storage.EXPECT().GetAllRoles().Return(nil, nil)

	_, err = s.dataStore.GetAllRoles(s.hasWriteCtx)
	s.NoError(err, "expected no error trying to read with permissions")
}

func (s *roleDataStoreTestSuite) TestEnforcesAdd() {
	s.storage.EXPECT().AddRole(gomock.Any()).Times(0)

	err := s.dataStore.AddRole(s.hasNoneCtx, &storage.Role{Name: "role"})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.AddRole(s.hasReadCtx, &storage.Role{Name: "role"})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *roleDataStoreTestSuite) TestAllowsAdd() {
	s.storage.EXPECT().AddRole(gomock.Any()).Return(nil)

	err := s.dataStore.AddRole(s.hasWriteCtx, &storage.Role{Name: "role"})
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *roleDataStoreTestSuite) TestEnforcesUpdate() {
	s.storage.EXPECT().AddRole(gomock.Any()).Times(0)

	err := s.dataStore.UpdateRole(s.hasNoneCtx, &storage.Role{Name: "role"})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.UpdateRole(s.hasReadCtx, &storage.Role{Name: "role"})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *roleDataStoreTestSuite) TestAllowsUpdate() {
	s.storage.EXPECT().UpdateRole(gomock.Any()).Return(nil)

	err := s.dataStore.UpdateRole(s.hasWriteCtx, &storage.Role{Name: "role"})
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *roleDataStoreTestSuite) TestEnforcesRemove() {
	s.storage.EXPECT().RemoveRole(gomock.Any()).Times(0)

	err := s.dataStore.RemoveRole(s.hasNoneCtx, "role")
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.RemoveRole(s.hasReadCtx, "role")
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *roleDataStoreTestSuite) TestAllowsRemove() {
	s.storage.EXPECT().RemoveRole(gomock.Any()).Return(nil)

	err := s.dataStore.RemoveRole(s.hasWriteCtx, "role")
	s.NoError(err, "expected no error trying to write with permissions")
}
