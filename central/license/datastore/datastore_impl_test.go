package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	storeMocks "github.com/stackrox/rox/central/license/internal/store/mocks"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

func TestLicenseDataStore(t *testing.T) {
	t.Parallel()
	if !features.ScopedAccessControl.Enabled() {
		t.Skip()
	}
	suite.Run(t, new(licenseDataStoreTestSuite))
}

type licenseDataStoreTestSuite struct {
	suite.Suite

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	dataStore DataStore
	storage   *storeMocks.MockStore

	mockCtrl *gomock.Controller
}

func (s *licenseDataStoreTestSuite) SetupTest() {
	s.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Group)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Group)))

	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)
	s.dataStore = New(s.storage)
}

func (s *licenseDataStoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *licenseDataStoreTestSuite) TestEnforcesList() {
	s.storage.EXPECT().ListLicenseKeys().Times(0)

	group, err := s.dataStore.ListLicenseKeys(s.hasNoneCtx)
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(group, "expected return value to be nil")
}

func (s *licenseDataStoreTestSuite) TestAllowsList() {
	s.storage.EXPECT().ListLicenseKeys().Return(nil, nil)

	_, err := s.dataStore.ListLicenseKeys(s.hasReadCtx)
	s.NoError(err, "expected no error trying to read with permissions")

	s.storage.EXPECT().ListLicenseKeys().Return(nil, nil)

	_, err = s.dataStore.ListLicenseKeys(s.hasReadCtx)
	s.NoError(err, "expected no error trying to read with permissions")
}

func (s *licenseDataStoreTestSuite) TestEnforcesUpsert() {
	s.storage.EXPECT().UpsertLicenseKeys(gomock.Any()).Times(0)

	err := s.dataStore.UpsertLicenseKeys(s.hasNoneCtx, []*storage.StoredLicenseKey{})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.UpsertLicenseKeys(s.hasNoneCtx, []*storage.StoredLicenseKey{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *licenseDataStoreTestSuite) TestAllowsUpsert() {
	s.storage.EXPECT().UpsertLicenseKeys(gomock.Any()).Return(nil)

	err := s.dataStore.UpsertLicenseKeys(s.hasWriteCtx, []*storage.StoredLicenseKey{})
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *licenseDataStoreTestSuite) TestEnforcesDelete() {
	s.storage.EXPECT().DeleteLicenseKey(gomock.Any()).Times(0)

	err := s.dataStore.DeleteLicenseKey(s.hasNoneCtx, "FAKEEID")
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.DeleteLicenseKey(s.hasNoneCtx, "FAKEID")
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *licenseDataStoreTestSuite) TestAllowsDelete() {
	s.storage.EXPECT().DeleteLicenseKey(gomock.Any()).Return(nil)

	err := s.dataStore.DeleteLicenseKey(s.hasWriteCtx, "FAKEID")
	s.NoError(err, "expected no error trying to write with permissions")
}
