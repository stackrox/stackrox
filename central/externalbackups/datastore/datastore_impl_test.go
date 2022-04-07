package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	storeMocks "github.com/stackrox/rox/central/externalbackups/internal/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

func TestExtBkpDataStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(extBkpDataStoreTestSuite))
}

const (
	FakeID   = "FAKEID"
	FakeName = "FAKENAME"
	FakeType = "FAKETYPE"
)

// NewFakeExtBkps constructs and returns a new External Backup object suitable for unit-testing.
func NewFakeExtBkp() *storage.ExternalBackup {
	return &storage.ExternalBackup{
		Id:   FakeID,
		Name: FakeName,
		Type: FakeType,
	}
}

// NewFakeListExtBkps constructs and returns a new slice of storage.ExternalBackup objects suitable for unit-testing.
func NewFakeListExtBkps() []*storage.ExternalBackup {
	return []*storage.ExternalBackup{
		NewFakeExtBkp(),
	}
}

type extBkpDataStoreTestSuite struct {
	suite.Suite

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	dataStore DataStore
	storage   *storeMocks.MockStore

	mockCtrl *gomock.Controller
}

func (s *extBkpDataStoreTestSuite) SetupTest() {
	s.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.BackupPlugins)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.BackupPlugins)))

	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)
	s.dataStore = New(s.storage)
}

func (s *extBkpDataStoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *extBkpDataStoreTestSuite) TestUpsertExtBkps() {
	s.storage.EXPECT().ListBackups().Times(0)

	result, err := s.dataStore.ListBackups(s.hasNoneCtx)
	s.NoError(err)
	s.Empty(result)

	s.storage.EXPECT().UpsertBackup(NewFakeExtBkp()).Return(nil)

	err = s.dataStore.UpsertBackup(s.hasWriteCtx, NewFakeExtBkp())
	s.NoError(err)

	s.storage.EXPECT().ListBackups().Return(NewFakeListExtBkps(), nil)

	bkps, err := s.dataStore.ListBackups(s.hasReadCtx)
	s.Equal(NewFakeListExtBkps(), bkps)
	s.NoError(err)
}

func (s *extBkpDataStoreTestSuite) TestEnforcesList() {
	s.storage.EXPECT().ListBackups().Times(0)

	result, err := s.dataStore.ListBackups(s.hasNoneCtx)
	s.NoError(err)
	s.Empty(result)
}

func (s *extBkpDataStoreTestSuite) TestAllowsList() {
	s.storage.EXPECT().ListBackups().Return(NewFakeListExtBkps(), nil)

	result, err := s.dataStore.ListBackups(s.hasReadCtx)
	s.NoError(err, "expected no error, should return nil without access")
	s.Equal(NewFakeListExtBkps(), result)
}

func (s *extBkpDataStoreTestSuite) TestEnforcesGet() {
	s.storage.EXPECT().GetBackup(gomock.Any()).Times(0)

	config, err := s.dataStore.GetBackup(s.hasNoneCtx, FakeID)
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(config, "expected return value to be nil")
}

func (s *extBkpDataStoreTestSuite) TestAllowsGet() {
	s.storage.EXPECT().GetBackup(gomock.Any()).Return(nil, nil)

	_, err := s.dataStore.GetBackup(s.hasReadCtx, FakeID)
	s.NoError(err, "expected no error trying to read with permissions")
}

func (s *extBkpDataStoreTestSuite) TestEnforcesUpsert() {
	s.storage.EXPECT().UpsertBackup(gomock.Any()).Times(0)

	err := s.dataStore.UpsertBackup(s.hasNoneCtx, &storage.ExternalBackup{})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.UpsertBackup(s.hasReadCtx, &storage.ExternalBackup{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *extBkpDataStoreTestSuite) TestAllowsUpsert() {
	s.storage.EXPECT().UpsertBackup(gomock.Any()).Return(nil)

	err := s.dataStore.UpsertBackup(s.hasWriteCtx, &storage.ExternalBackup{})
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *extBkpDataStoreTestSuite) TestEnforcesRemove() {
	s.storage.EXPECT().RemoveBackup(gomock.Any()).Times(0)

	err := s.dataStore.RemoveBackup(s.hasNoneCtx, FakeID)
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.RemoveBackup(s.hasReadCtx, FakeID)
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *extBkpDataStoreTestSuite) TestAllowsRemove() {
	s.storage.EXPECT().RemoveBackup(gomock.Any()).Return(nil)

	err := s.dataStore.RemoveBackup(s.hasWriteCtx, FakeID)
	s.NoError(err, "expected no error trying to write with permissions")
}
