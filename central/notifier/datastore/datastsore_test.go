package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	storeMocks "github.com/stackrox/rox/central/notifier/datastore/internal/store/mocks"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

func TestNotifierDataStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(notifierDataStoreTestSuite))
}

type notifierDataStoreTestSuite struct {
	suite.Suite

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	dataStore DataStore
	storage   *storeMocks.MockStore

	mockCtrl *gomock.Controller
}

func (s *notifierDataStoreTestSuite) SetupTest() {
	s.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Notifier)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Notifier)))

	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)
	s.dataStore = New(s.storage)
}

func (s *notifierDataStoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *notifierDataStoreTestSuite) TestEnforcesGet() {
	s.storage.EXPECT().GetNotifier(gomock.Any()).Times(0)

	notifier, exists, err := s.dataStore.GetNotifier(s.hasNoneCtx, "notifier")
	s.NoError(err, "expected no error, should return nil without access")
	s.False(exists, "expected exists to be set to false")
	s.Nil(notifier, "expected return value to be nil")
}

func (s *notifierDataStoreTestSuite) TestAllowsGet() {
	s.storage.EXPECT().GetNotifier(gomock.Any()).Return(nil, true, nil)

	_, exists, err := s.dataStore.GetNotifier(s.hasReadCtx, "notifier")
	s.NoError(err, "expected no error trying to read with permissions")
	s.True(exists, "expected exists to be set to false")

	s.storage.EXPECT().GetNotifier(gomock.Any()).Return(nil, true, nil)

	_, exists, err = s.dataStore.GetNotifier(s.hasWriteCtx, "notifier")
	s.NoError(err, "expected no error trying to read with permissions")
	s.True(exists, "expected exists to be set to false")
}

func (s *notifierDataStoreTestSuite) TestEnforcesGetMany() {
	s.storage.EXPECT().GetNotifiers(gomock.Any()).Times(0)

	notifiers, err := s.dataStore.GetNotifiers(s.hasNoneCtx, &v1.GetNotifiersRequest{})
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(notifiers, "expected return value to be nil")
}

func (s *notifierDataStoreTestSuite) TestAllowsGetMany() {
	s.storage.EXPECT().GetNotifiers(gomock.Any()).Return(nil, nil)

	_, err := s.dataStore.GetNotifiers(s.hasReadCtx, &v1.GetNotifiersRequest{})
	s.NoError(err, "expected no error trying to read with permissions")

	s.storage.EXPECT().GetNotifiers(gomock.Any()).Return(nil, nil)

	_, err = s.dataStore.GetNotifiers(s.hasWriteCtx, &v1.GetNotifiersRequest{})
	s.NoError(err, "expected no error trying to read with permissions")
}

func (s *notifierDataStoreTestSuite) TestEnforcesAdd() {
	s.storage.EXPECT().AddNotifier(gomock.Any()).Times(0)

	_, err := s.dataStore.AddNotifier(s.hasNoneCtx, &storage.Notifier{})
	s.Error(err, "expected an error trying to write without permissions")

	_, err = s.dataStore.AddNotifier(s.hasReadCtx, &storage.Notifier{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *notifierDataStoreTestSuite) TestAllowsAdd() {
	s.storage.EXPECT().AddNotifier(gomock.Any()).Return("", nil)

	_, err := s.dataStore.AddNotifier(s.hasWriteCtx, &storage.Notifier{})
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *notifierDataStoreTestSuite) TestEnforcesUpdate() {
	s.storage.EXPECT().UpdateNotifier(gomock.Any()).Times(0)

	err := s.dataStore.UpdateNotifier(s.hasNoneCtx, &storage.Notifier{})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.UpdateNotifier(s.hasReadCtx, &storage.Notifier{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *notifierDataStoreTestSuite) TestAllowsUpdate() {
	s.storage.EXPECT().UpdateNotifier(gomock.Any()).Return(nil)

	err := s.dataStore.UpdateNotifier(s.hasWriteCtx, &storage.Notifier{})
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *notifierDataStoreTestSuite) TestEnforcesRemove() {
	s.storage.EXPECT().RemoveNotifier(gomock.Any()).Times(0)

	err := s.dataStore.RemoveNotifier(s.hasNoneCtx, "notifier")
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.RemoveNotifier(s.hasReadCtx, "notifier")
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *notifierDataStoreTestSuite) TestAllowsRemove() {
	s.storage.EXPECT().RemoveNotifier(gomock.Any()).Return(nil)

	err := s.dataStore.RemoveNotifier(s.hasWriteCtx, "notifier")
	s.NoError(err, "expected no error trying to write with permissions")
}
