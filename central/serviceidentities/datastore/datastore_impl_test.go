package datastore

import (
	"context"
	"testing"

	storeMocks "github.com/stackrox/rox/central/serviceidentities/internal/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestServiceIdentityDataStore(t *testing.T) {
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
			sac.ResourceScopeKeys(resources.Administration)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))
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

	s.storage.EXPECT().Upsert(gomock.Any(), srvID).Return(nil).Times(1)

	err := s.dataStore.AddServiceIdentity(s.hasWriteCtx, srvID)
	s.NoError(err)
}

func (s *serviceIdentityDataStoreTestSuite) TestEnforcesGet() {
	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).Times(0)

	err := s.dataStore.ForEachServiceIdentity(s.hasNoneCtx, nil)
	s.NoError(err, "expected no error, should return nil without access")
}

func (s *serviceIdentityDataStoreTestSuite) TestAllowsGet() {
	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := s.dataStore.ForEachServiceIdentity(s.hasReadCtx, nil)
	s.NoError(err, "expected no error trying to read with permissions")
}

func (s *serviceIdentityDataStoreTestSuite) TestEnforcesAdd() {
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Times(0)

	err := s.dataStore.AddServiceIdentity(s.hasNoneCtx, &storage.ServiceIdentity{})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.AddServiceIdentity(s.hasReadCtx, &storage.ServiceIdentity{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *serviceIdentityDataStoreTestSuite) TestAllowsAdd() {
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := s.dataStore.AddServiceIdentity(s.hasWriteCtx, &storage.ServiceIdentity{})
	s.NoError(err, "expected no error trying to write with permissions")
}
