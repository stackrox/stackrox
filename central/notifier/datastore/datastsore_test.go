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
)

func TestNotifierDataStore(t *testing.T) {
	t.Parallel()
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
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Integration)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
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
	s.storage.EXPECT().GetAll(gomock.Any()).Return(nil, nil).Times(1)

	_, err := s.dataStore.GetNotifiers(s.hasReadCtx)
	s.NoError(err, "expected no error trying to read with permissions")

	s.storage.EXPECT().GetAll(gomock.Any()).Return(nil, nil).Times(1)

	_, err = s.dataStore.GetNotifiers(s.hasWriteCtx)
	s.NoError(err, "expected no error trying to read with permissions")
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

	err := s.dataStore.UpdateNotifier(s.hasNoneCtx, &storage.Notifier{Id: "id"})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.UpdateNotifier(s.hasReadCtx, &storage.Notifier{Id: "id"})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *notifierDataStoreTestSuite) TestAllowsUpdate() {
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, true, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := s.dataStore.UpdateNotifier(s.hasWriteCtx, &storage.Notifier{Id: "id"})
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *notifierDataStoreTestSuite) TestErrorOnUpdate() {
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, false, nil).Times(1)

	err := s.dataStore.UpdateNotifier(s.hasWriteCtx, &storage.Notifier{Id: "id"})
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
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&storage.Notifier{
		Id:   "id",
		Name: "name",
		Traits: &storage.Traits{
			MutabilityMode: storage.Traits_ALLOW_MUTATE,
		},
	}, true, nil).Times(1)

	err := s.dataStore.UpdateNotifier(s.hasWriteCtx, &storage.Notifier{
		Id:   "id",
		Name: "new name",
		Traits: &storage.Traits{
			MutabilityMode: storage.Traits_ALLOW_MUTATE_FORCED,
		},
	})
	s.ErrorIs(err, errox.InvalidArgs)
}

func (s *notifierDataStoreTestSuite) TestRemoveDeclarativeViaAPI() {
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&storage.Notifier{
		Id:   "id",
		Name: "name",
		Traits: &storage.Traits{
			Origin: storage.Traits_DECLARATIVE,
		},
	}, true, nil).Times(1)

	err := s.dataStore.RemoveNotifier(s.hasWriteCtx, "id")
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *notifierDataStoreTestSuite) TestRemoveDeclarativeSuccess() {
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&storage.Notifier{
		Id:   "id",
		Name: "name",
		Traits: &storage.Traits{
			Origin: storage.Traits_DECLARATIVE,
		},
	}, true, nil).Times(1)
	s.storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := s.dataStore.RemoveNotifier(s.hasWriteDeclarativeCtx, "id")
	s.NoError(err)
}

func (s *notifierDataStoreTestSuite) TestUpdateDeclarativeViaAPI() {
	ap := &storage.Notifier{
		Id:   "id",
		Name: "name",
		Traits: &storage.Traits{
			Origin: storage.Traits_DECLARATIVE,
		},
	}
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(ap, true, nil).Times(1)

	err := s.dataStore.UpdateNotifier(s.hasWriteCtx, ap)
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *notifierDataStoreTestSuite) TestUpdateDeclarativeSuccess() {
	ap := &storage.Notifier{
		Id:   "id",
		Name: "name",
		Traits: &storage.Traits{
			Origin: storage.Traits_DECLARATIVE,
		},
	}
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(ap, true, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := s.dataStore.UpdateNotifier(s.hasWriteDeclarativeCtx, ap)
	s.NoError(err)
}

func (s *notifierDataStoreTestSuite) TestRemoveImperativeDeclaratively() {
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&storage.Notifier{
		Id:   "id",
		Name: "name",
		Traits: &storage.Traits{
			Origin: storage.Traits_IMPERATIVE,
		},
	}, true, nil).Times(1)

	err := s.dataStore.RemoveNotifier(s.hasWriteDeclarativeCtx, "id")
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *notifierDataStoreTestSuite) TestUpdateImperativeDeclaratively() {
	ap := &storage.Notifier{
		Id:   "id",
		Name: "name",
		Traits: &storage.Traits{
			Origin: storage.Traits_IMPERATIVE,
		},
	}
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(ap, true, nil).Times(1)

	err := s.dataStore.UpdateNotifier(s.hasWriteDeclarativeCtx, ap)
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *notifierDataStoreTestSuite) TestAddDeclarativeViaAPI() {
	ap := &storage.Notifier{
		Id:   "id",
		Name: "name",
		Traits: &storage.Traits{
			Origin: storage.Traits_DECLARATIVE,
		},
	}

	_, err := s.dataStore.AddNotifier(s.hasWriteCtx, ap)
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *notifierDataStoreTestSuite) TestAddDeclarativeSuccess() {
	ap := &storage.Notifier{
		Id:   "id",
		Name: "name",
		Traits: &storage.Traits{
			Origin: storage.Traits_DECLARATIVE,
		},
	}
	s.storage.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(false, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	_, err := s.dataStore.AddNotifier(s.hasWriteDeclarativeCtx, ap)
	s.NoError(err)
}

func (s *notifierDataStoreTestSuite) TestAddImperativeDeclaratively() {
	ap := &storage.Notifier{
		Id:   "id",
		Name: "name",
		Traits: &storage.Traits{
			Origin: storage.Traits_IMPERATIVE,
		},
	}

	_, err := s.dataStore.AddNotifier(s.hasWriteDeclarativeCtx, ap)
	s.ErrorIs(err, errox.NotAuthorized)
}
