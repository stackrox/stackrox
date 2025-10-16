package datastore

import (
	"context"
	"testing"

	storeMocks "github.com/stackrox/rox/central/notifier/datastore/internal/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
)

func TestNotifierDataStore(t *testing.T) {
	suite.Run(t, new(notifierDataStoreTestSuite))
}

type notifierDataStoreTestSuite struct {
	suite.Suite

	hasNoneCtx             context.Context
	hasReadCtx             context.Context
	hasWriteCtx            context.Context
	hasWriteDeclarativeCtx context.Context

	dataStore DataStore
	storage   *storeMocks.MockStore

	mockCtrl *gomock.Controller
}

func (s *notifierDataStoreTestSuite) SetupTest() {
	s.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Integration)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Integration)))
	s.hasWriteDeclarativeCtx = declarativeconfig.WithModifyDeclarativeResource(s.hasWriteCtx)
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
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, true, nil).Times(1)

	_, exists, err := s.dataStore.GetNotifier(s.hasReadCtx, "notifier")
	s.NoError(err, "expected no error trying to read with permissions")
	s.True(exists, "expected exists to be set to false")

	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, true, nil).Times(1)

	_, exists, err = s.dataStore.GetNotifier(s.hasWriteCtx, "notifier")
	s.NoError(err, "expected no error trying to read with permissions")
	s.True(exists, "expected exists to be set to false")
}

func (s *notifierDataStoreTestSuite) TestExists() {
	exists, err := s.dataStore.Exists(s.hasNoneCtx, "notifier")
	s.NoError(err, "expected no error, should return nil without access")
	s.False(exists, "expected exists to be set to false")

	s.storage.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, nil).Times(1)

	exists, err = s.dataStore.Exists(s.hasReadCtx, "notifier")
	s.NoError(err, "expected no error trying to read with permissions")
	s.False(exists, "expected exists to be set to false")

	s.storage.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(true, nil).Times(1)

	exists, err = s.dataStore.Exists(s.hasReadCtx, "notifier")
	s.NoError(err, "expected no error trying to read with permissions")
	s.True(exists, "expected exists to be set to true")
}

func (s *notifierDataStoreTestSuite) TestGetScrubbedNotifier() {
	generic := &storage.Generic{}
	generic.SetPassword("test")
	testNotifier := &storage.Notifier{}
	testNotifier.SetGeneric(proto.ValueOrDefault(generic))

	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(testNotifier, true, nil).Times(1)

	scrubbedNotifier, exists, err := s.dataStore.GetScrubbedNotifier(s.hasReadCtx, "test")
	s.NoError(err, "expected no error trying to read with permissions")
	s.True(exists, "expected exists to be set to false")
	s.Equal("******", scrubbedNotifier.GetConfig().(*storage.Notifier_Generic).Generic.GetPassword())
}

func (s *notifierDataStoreTestSuite) TestEnforcesForEach() {
	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).Times(0)

	err := s.dataStore.ForEachNotifier(s.hasNoneCtx, nil)
	s.NoError(err, "expected no error, should return nil without access")
}

func (s *notifierDataStoreTestSuite) TestAllowsForEach() {
	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := s.dataStore.ForEachNotifier(s.hasReadCtx, nil)
	s.NoError(err, "expected no error trying to read with permissions")
}

func (s *notifierDataStoreTestSuite) TestGetScrubbedNotifiers() {
	generic := &storage.Generic{}
	generic.SetPassword("test")
	testNotifier := &storage.Notifier{}
	testNotifier.SetGeneric(proto.ValueOrDefault(generic))

	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, fn func(obj *storage.Notifier) error) error {
			return fn(testNotifier)
		}).Times(1)

	err := s.dataStore.ForEachScrubbedNotifier(s.hasReadCtx, func(scrubbedNotifier *storage.Notifier) error {
		s.Equal("******", scrubbedNotifier.GetConfig().(*storage.Notifier_Generic).Generic.GetPassword())
		return nil
	})
	s.NoError(err, "expected no error trying to read with permissions")
}

func (s *notifierDataStoreTestSuite) TestEnforcesAdd() {
	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).Times(0)

	_, err := s.dataStore.AddNotifier(s.hasNoneCtx, &storage.Notifier{})
	s.Error(err, "expected an error trying to write without permissions")

	_, err = s.dataStore.AddNotifier(s.hasReadCtx, &storage.Notifier{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *notifierDataStoreTestSuite) TestAllowsAdd() {
	s.storage.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	_, err := s.dataStore.AddNotifier(s.hasWriteCtx, &storage.Notifier{})
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *notifierDataStoreTestSuite) TestErrorOnAdd() {
	s.storage.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(true, nil)

	_, err := s.dataStore.AddNotifier(s.hasWriteCtx, &storage.Notifier{})
	s.Error(err)
}

func (s *notifierDataStoreTestSuite) TestEnforcesUpdate() {
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Times(0)

	notifier := &storage.Notifier{}
	notifier.SetId("id")
	err := s.dataStore.UpdateNotifier(s.hasNoneCtx, notifier)
	s.Error(err, "expected an error trying to write without permissions")

	notifier2 := &storage.Notifier{}
	notifier2.SetId("id")
	err = s.dataStore.UpdateNotifier(s.hasReadCtx, notifier2)
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *notifierDataStoreTestSuite) TestAllowsUpdate() {
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, true, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	notifier := &storage.Notifier{}
	notifier.SetId("id")
	err := s.dataStore.UpdateNotifier(s.hasWriteCtx, notifier)
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *notifierDataStoreTestSuite) TestErrorOnUpdate() {
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, false, nil).Times(1)

	notifier := &storage.Notifier{}
	notifier.SetId("id")
	err := s.dataStore.UpdateNotifier(s.hasWriteCtx, notifier)
	s.Error(err)
}

func (s *notifierDataStoreTestSuite) TestEnforcesRemove() {
	s.storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)

	err := s.dataStore.RemoveNotifier(s.hasNoneCtx, "notifier")
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.RemoveNotifier(s.hasReadCtx, "notifier")
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *notifierDataStoreTestSuite) TestAllowsRemove() {
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, true, nil).Times(1)
	s.storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := s.dataStore.RemoveNotifier(s.hasWriteCtx, "notifier")
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *notifierDataStoreTestSuite) TestUpdateMutableToImmutable() {
	traits := &storage.Traits{}
	traits.SetMutabilityMode(storage.Traits_ALLOW_MUTATE)
	notifier := &storage.Notifier{}
	notifier.SetId("id")
	notifier.SetName("name")
	notifier.SetTraits(traits)
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(notifier, true, nil).Times(1)

	traits2 := &storage.Traits{}
	traits2.SetMutabilityMode(storage.Traits_ALLOW_MUTATE_FORCED)
	notifier2 := &storage.Notifier{}
	notifier2.SetId("id")
	notifier2.SetName("new name")
	notifier2.SetTraits(traits2)
	err := s.dataStore.UpdateNotifier(s.hasWriteCtx, notifier2)
	s.ErrorIs(err, errox.InvalidArgs)
}

func (s *notifierDataStoreTestSuite) TestRemoveDeclarativeViaAPI() {
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_DECLARATIVE)
	notifier := &storage.Notifier{}
	notifier.SetId("id")
	notifier.SetName("name")
	notifier.SetTraits(traits)
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(notifier, true, nil).Times(1)

	err := s.dataStore.RemoveNotifier(s.hasWriteCtx, "id")
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *notifierDataStoreTestSuite) TestRemoveDeclarativeSuccess() {
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_DECLARATIVE)
	notifier := &storage.Notifier{}
	notifier.SetId("id")
	notifier.SetName("name")
	notifier.SetTraits(traits)
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(notifier, true, nil).Times(1)
	s.storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := s.dataStore.RemoveNotifier(s.hasWriteDeclarativeCtx, "id")
	s.NoError(err)
}

func (s *notifierDataStoreTestSuite) TestUpdateDeclarativeViaAPI() {
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_DECLARATIVE)
	ap := &storage.Notifier{}
	ap.SetId("id")
	ap.SetName("name")
	ap.SetTraits(traits)
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(ap, true, nil).Times(1)

	err := s.dataStore.UpdateNotifier(s.hasWriteCtx, ap)
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *notifierDataStoreTestSuite) TestUpdateDeclarativeSuccess() {
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_DECLARATIVE)
	ap := &storage.Notifier{}
	ap.SetId("id")
	ap.SetName("name")
	ap.SetTraits(traits)
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(ap, true, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := s.dataStore.UpdateNotifier(s.hasWriteDeclarativeCtx, ap)
	s.NoError(err)
}

func (s *notifierDataStoreTestSuite) TestRemoveImperativeDeclaratively() {
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_IMPERATIVE)
	notifier := &storage.Notifier{}
	notifier.SetId("id")
	notifier.SetName("name")
	notifier.SetTraits(traits)
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(notifier, true, nil).Times(1)

	err := s.dataStore.RemoveNotifier(s.hasWriteDeclarativeCtx, "id")
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *notifierDataStoreTestSuite) TestUpdateImperativeDeclaratively() {
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_IMPERATIVE)
	ap := &storage.Notifier{}
	ap.SetId("id")
	ap.SetName("name")
	ap.SetTraits(traits)
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(ap, true, nil).Times(1)

	err := s.dataStore.UpdateNotifier(s.hasWriteDeclarativeCtx, ap)
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *notifierDataStoreTestSuite) TestAddDeclarativeViaAPI() {
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_DECLARATIVE)
	ap := &storage.Notifier{}
	ap.SetId("id")
	ap.SetName("name")
	ap.SetTraits(traits)

	_, err := s.dataStore.AddNotifier(s.hasWriteCtx, ap)
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *notifierDataStoreTestSuite) TestAddDeclarativeSuccess() {
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_DECLARATIVE)
	ap := &storage.Notifier{}
	ap.SetId("id")
	ap.SetName("name")
	ap.SetTraits(traits)
	s.storage.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	_, err := s.dataStore.AddNotifier(s.hasWriteDeclarativeCtx, ap)
	s.NoError(err)
}

func (s *notifierDataStoreTestSuite) TestAddImperativeDeclaratively() {
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_IMPERATIVE)
	ap := &storage.Notifier{}
	ap.SetId("id")
	ap.SetName("name")
	ap.SetTraits(traits)

	_, err := s.dataStore.AddNotifier(s.hasWriteDeclarativeCtx, ap)
	s.ErrorIs(err, errox.NotAuthorized)
}
