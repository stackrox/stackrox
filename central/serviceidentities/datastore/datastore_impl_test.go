package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	storeMocks "github.com/stackrox/rox/central/serviceidentities/internal/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

func TestServiceIdentityDataStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(serviceIdentityDataStoreTestSuite))
}

type serviceIdentityDataStoreTestSuite struct {
	suite.Suite

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	dataStore DataStore
	storage   *storeMocks.MockStore

	mockCtrl *gomock.Controller
}

func (s *serviceIdentityDataStoreTestSuite) SetupTest() {
	s.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.ServiceIdentity)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.ServiceIdentity)))

	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)
	s.dataStore = New(s.storage)
}

func (s *serviceIdentityDataStoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *serviceIdentityDataStoreTestSuite) TestAddSrvId() {
	srvID := &storage.ServiceIdentity{
		Id: "FAKEID",
	}
	allSrvIDs := []*storage.ServiceIdentity{srvID}

	s.storage.EXPECT().GetServiceIdentities().Return(allSrvIDs, nil)
	s.storage.EXPECT().AddServiceIdentity(srvID).Return(nil)

	err := s.dataStore.AddServiceIdentity(s.hasWriteCtx, srvID)
	s.NoError(err)

	result, err := s.dataStore.GetServiceIdentities(s.hasReadCtx)
	s.Equal(allSrvIDs, result)
	s.NoError(err)
}

func (s *serviceIdentityDataStoreTestSuite) TestEnforcesGet() {
	s.storage.EXPECT().GetServiceIdentities().Times(0)

	group, err := s.dataStore.GetServiceIdentities(s.hasNoneCtx)
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(group, "expected return value to be nil")
}

func (s *serviceIdentityDataStoreTestSuite) TestAllowsGet() {
	s.storage.EXPECT().GetServiceIdentities().Return(nil, nil)

	_, err := s.dataStore.GetServiceIdentities(s.hasReadCtx)
	s.NoError(err, "expected no error trying to read with permissions")
}

func (s *serviceIdentityDataStoreTestSuite) TestEnforcesAdd() {
	s.storage.EXPECT().AddServiceIdentity(gomock.Any()).Times(0)

	err := s.dataStore.AddServiceIdentity(s.hasNoneCtx, &storage.ServiceIdentity{})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.AddServiceIdentity(s.hasReadCtx, &storage.ServiceIdentity{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *serviceIdentityDataStoreTestSuite) TestAllowsAdd() {
	s.storage.EXPECT().AddServiceIdentity(gomock.Any()).Return(nil)

	err := s.dataStore.AddServiceIdentity(s.hasWriteCtx, &storage.ServiceIdentity{})
	s.NoError(err, "expected no error trying to write with permissions")
}
