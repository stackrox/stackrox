package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	storeMocks "github.com/stackrox/rox/central/notifier/datastore/internal/store/mocks"
	"github.com/stackrox/rox/central/role/resources"
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

	hasReadIntegrationsCtx  context.Context
	hasWriteIntegrationsCtx context.Context

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
	s.hasReadIntegrationsCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Integration)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Notifier)))
	s.hasWriteIntegrationsCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Integration)))
	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)
	s.dataStore = New(s.storage)
}

func (s *notifierDataStoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *notifierDataStoreTestSuite) TestEnforcesGet() {
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Times(0)

	notifier, exists, err := s.dataStore.GetNotifier(s.hasNoneCtx, "notifier")
	s.NoError(err, "expected no error, should return nil without access")
	s.False(exists, "expected exists to be set to false")
	s.Nil(notifier, "expected return value to be nil")
}

func (s *notifierDataStoreTestSuite) TestAllowsGet() {
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, true, nil).Times(2)

	_, exists, err := s.dataStore.GetNotifier(s.hasReadCtx, "notifier")
	s.NoError(err, "expected no error trying to read with permissions")
	s.True(exists, "expected exists to be set to false")

	_, exists, err = s.dataStore.GetNotifier(s.hasReadIntegrationsCtx, "notifier")
	s.NoError(err, "expected no error trying to read with permissions")
	s.True(exists, "expected exists to be set to false")

	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, true, nil).Times(2)

	_, exists, err = s.dataStore.GetNotifier(s.hasWriteCtx, "notifier")
	s.NoError(err, "expected no error trying to read with permissions")
	s.True(exists, "expected exists to be set to false")

	_, exists, err = s.dataStore.GetNotifier(s.hasWriteIntegrationsCtx, "notifier")
	s.NoError(err, "expected no error trying to read with Integration permissions")
	s.True(exists, "expected exists to be set to false")
}

func (s *notifierDataStoreTestSuite) TestGetScrubbedNotifier() {
	testNotifier := &storage.Notifier{
		Config: &storage.Notifier_Generic{
			Generic: &storage.Generic{
				Password: "test",
			},
		},
	}

	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(testNotifier, true, nil).Times(1)

	scrubbedNotifier, exists, err := s.dataStore.GetScrubbedNotifier(s.hasReadCtx, "test")
	s.NoError(err, "expected no error trying to read with permissions")
	s.True(exists, "expected exists to be set to false")
	s.Equal("******", scrubbedNotifier.Config.(*storage.Notifier_Generic).Generic.Password)
}

func (s *notifierDataStoreTestSuite) TestEnforcesGetMany() {
	s.storage.EXPECT().GetAll(gomock.Any()).Times(0)

	notifiers, err := s.dataStore.GetNotifiers(s.hasNoneCtx)
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(notifiers, "expected return value to be nil")
}

func (s *notifierDataStoreTestSuite) TestAllowsGetMany() {
	s.storage.EXPECT().GetAll(gomock.Any()).Return(nil, nil).Times(2)

	_, err := s.dataStore.GetNotifiers(s.hasReadCtx)
	s.NoError(err, "expected no error trying to read with permissions")

	_, err = s.dataStore.GetNotifiers(s.hasReadIntegrationsCtx)
	s.NoError(err, "expected no error trying to read with permissions")

	s.storage.EXPECT().GetAll(gomock.Any()).Return(nil, nil).Times(2)

	_, err = s.dataStore.GetNotifiers(s.hasWriteCtx)
	s.NoError(err, "expected no error trying to read with permissions")

	_, err = s.dataStore.GetNotifiers(s.hasWriteIntegrationsCtx)
	s.NoError(err, "expected no error trying to read with Integration permissions")
}

func (s *notifierDataStoreTestSuite) TestGetScrubbedNotifiers() {
	testNotifier := &storage.Notifier{
		Config: &storage.Notifier_Generic{
			Generic: &storage.Generic{
				Password: "test",
			},
		},
	}

	s.storage.EXPECT().GetAll(gomock.Any()).Return([]*storage.Notifier{testNotifier}, nil).Times(1)

	scrubbedNotifiers, err := s.dataStore.GetScrubbedNotifiers(s.hasReadCtx)
	s.NoError(err, "expected no error trying to read with permissions")
	s.Equal(1, len(scrubbedNotifiers), "expected one notifier in the list")
	s.Equal("******", scrubbedNotifiers[0].Config.(*storage.Notifier_Generic).Generic.Password)
}

func (s *notifierDataStoreTestSuite) TestEnforcesAdd() {
	s.storage.EXPECT().GetAll(gomock.Any()).Times(0)

	_, err := s.dataStore.AddNotifier(s.hasNoneCtx, &storage.Notifier{})
	s.Error(err, "expected an error trying to write without permissions")

	_, err = s.dataStore.AddNotifier(s.hasReadCtx, &storage.Notifier{})
	s.Error(err, "expected an error trying to write without permissions")

	_, err = s.dataStore.AddNotifier(s.hasReadIntegrationsCtx, &storage.Notifier{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *notifierDataStoreTestSuite) TestAllowsAdd() {
	s.storage.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, nil).Times(2)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(2)

	_, err := s.dataStore.AddNotifier(s.hasWriteCtx, &storage.Notifier{})
	s.NoError(err, "expected no error trying to write with permissions")

	_, err = s.dataStore.AddNotifier(s.hasWriteIntegrationsCtx, &storage.Notifier{})
	s.NoError(err, "expected no error trying to write with Integration permissions")
}

func (s *notifierDataStoreTestSuite) TestErrorOnAdd() {
	s.storage.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(true, nil)

	_, err := s.dataStore.AddNotifier(s.hasWriteCtx, &storage.Notifier{})
	s.Error(err)
}

func (s *notifierDataStoreTestSuite) TestEnforcesUpdate() {
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Times(0)

	err := s.dataStore.UpdateNotifier(s.hasNoneCtx, &storage.Notifier{Id: "id"})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.UpdateNotifier(s.hasReadCtx, &storage.Notifier{Id: "id"})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.UpdateNotifier(s.hasReadIntegrationsCtx, &storage.Notifier{Id: "id"})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *notifierDataStoreTestSuite) TestAllowsUpdate() {
	s.storage.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(true, nil).Times(2)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(2)

	err := s.dataStore.UpdateNotifier(s.hasWriteCtx, &storage.Notifier{Id: "id"})
	s.NoError(err, "expected no error trying to write with permissions")

	err = s.dataStore.UpdateNotifier(s.hasWriteIntegrationsCtx, &storage.Notifier{})
	s.NoError(err, "expected no error trying to write with Integration permissions")
}

func (s *notifierDataStoreTestSuite) TestErrorOnUpdate() {
	s.storage.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, nil)

	err := s.dataStore.UpdateNotifier(s.hasWriteCtx, &storage.Notifier{Id: "id"})
	s.Error(err)
}

func (s *notifierDataStoreTestSuite) TestEnforcesRemove() {
	s.storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)

	err := s.dataStore.RemoveNotifier(s.hasNoneCtx, "notifier")
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.RemoveNotifier(s.hasReadCtx, "notifier")
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.RemoveNotifier(s.hasReadIntegrationsCtx, "notifier")
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *notifierDataStoreTestSuite) TestAllowsRemove() {
	s.storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(2)

	err := s.dataStore.RemoveNotifier(s.hasWriteCtx, "notifier")
	s.NoError(err, "expected no error trying to write with permissions")

	err = s.dataStore.RemoveNotifier(s.hasWriteIntegrationsCtx, "notifier")
	s.NoError(err, "expected no error trying to write with Integration permissions")
}
