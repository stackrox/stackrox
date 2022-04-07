package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	storeMocks "github.com/stackrox/rox/central/sensorupgradeconfig/datastore/internal/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
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
			sac.ResourceScopeKeys(resources.SensorUpgradeConfig)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.SensorUpgradeConfig)))

	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)
	s.storage.EXPECT().GetSensorUpgradeConfig().Return(nil, nil)
	var err error
	s.dataStore, err = New(s.storage)
	s.Require().NoError(err)
}

func (s *sensorUpgradeConfigDataStoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *sensorUpgradeConfigDataStoreTestSuite) TestEnforcesGet() {
	s.storage.EXPECT().GetSensorUpgradeConfig().Times(0)

	config, err := s.dataStore.GetSensorUpgradeConfig(s.hasNoneCtx)
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(config, "expected return value to be nil")
}

func (s *sensorUpgradeConfigDataStoreTestSuite) TestAllowsGet() {
	s.storage.EXPECT().GetSensorUpgradeConfig().Return(nil, nil)

	_, err := s.dataStore.GetSensorUpgradeConfig(s.hasReadCtx)
	s.NoError(err, "expected no error trying to read with permissions")

	s.storage.EXPECT().GetSensorUpgradeConfig().Return(nil, nil)

	_, err = s.dataStore.GetSensorUpgradeConfig(s.hasWriteCtx)
	s.NoError(err, "expected no error trying to read with permissions")
}

func (s *sensorUpgradeConfigDataStoreTestSuite) TestEnforcesUpdate() {
	s.storage.EXPECT().UpsertSensorUpgradeConfig(gomock.Any()).Times(0)

	err := s.dataStore.UpsertSensorUpgradeConfig(s.hasNoneCtx, &storage.SensorUpgradeConfig{})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.UpsertSensorUpgradeConfig(s.hasReadCtx, &storage.SensorUpgradeConfig{})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *sensorUpgradeConfigDataStoreTestSuite) TestAllowsUpdate() {
	s.storage.EXPECT().UpsertSensorUpgradeConfig(gomock.Any()).Return(nil)

	err := s.dataStore.UpsertSensorUpgradeConfig(s.hasWriteCtx, &storage.SensorUpgradeConfig{})
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *sensorUpgradeConfigDataStoreTestSuite) TestDefault() {
	s.storage.EXPECT().GetSensorUpgradeConfig().Return(nil, nil)
	s.storage.EXPECT().UpsertSensorUpgradeConfig(defaultConfig).Return(nil)

	s.Require().NoError(addDefaultConfigIfEmpty(s.dataStore))
}
