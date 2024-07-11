package datastore

import (
	"context"
	"testing"

	storeMocks "github.com/stackrox/rox/central/sensorupgradeconfig/datastore/internal/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestSensorUpgradeConfigDataStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(sensorUpgradeConfigDataStoreTestSuite))
}

type sensorUpgradeConfigDataStoreTestSuite struct {
	suite.Suite

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	dataStore DataStore
	storage   *storeMocks.MockStore

	mockCtrl *gomock.Controller
}

func (s *sensorUpgradeConfigDataStoreTestSuite) SetupTest() {
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

func (s *sensorUpgradeConfigDataStoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *sensorUpgradeConfigDataStoreTestSuite) TestEnforcesGet() {
	s.storage.EXPECT().Get(gomock.Any()).Times(0)

	config, err := s.dataStore.GetSensorUpgradeConfig(s.hasNoneCtx)
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(config, "expected return value to be nil")
}

func (s *sensorUpgradeConfigDataStoreTestSuite) TestAllowsGet() {
	s.storage.EXPECT().Get(gomock.Any()).Return(nil, false, nil)

	_, err := s.dataStore.GetSensorUpgradeConfig(s.hasReadCtx)
	s.NoError(err, "expected no error trying to read with permissions")

	s.storage.EXPECT().Get(gomock.Any()).Return(nil, false, nil).Times(1)

	_, err = s.dataStore.GetSensorUpgradeConfig(s.hasWriteCtx)
	s.NoError(err, "expected no error trying to read with permissions")

}

func (s *sensorUpgradeConfigDataStoreTestSuite) TestEnforcesUpdate() {
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Times(0)

	err := s.dataStore.UpsertSensorUpgradeConfig(s.hasNoneCtx, &storage.SensorUpgradeConfig{})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.UpsertSensorUpgradeConfig(s.hasReadCtx, &storage.SensorUpgradeConfig{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *sensorUpgradeConfigDataStoreTestSuite) TestAllowsUpdate() {
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := s.dataStore.UpsertSensorUpgradeConfig(s.hasWriteCtx, &storage.SensorUpgradeConfig{})
	s.NoError(err, "expected no error trying to write with permissions")

}
