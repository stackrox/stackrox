package service

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	managerMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	datastoreMocks "github.com/stackrox/rox/central/sensorupgradeconfig/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type SensorUpgradeServiceTestSuite struct {
	suite.Suite
	mockCtrl  *gomock.Controller
	dataStore *datastoreMocks.MockDataStore
	manager   *managerMocks.MockManager
}

func TestSensorUpgradeService(t *testing.T) {
	suite.Run(t, new(SensorUpgradeServiceTestSuite))
}

var _ suite.SetupTestSuite = (*SensorUpgradeServiceTestSuite)(nil)

func (s *SensorUpgradeServiceTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.dataStore = datastoreMocks.NewMockDataStore(s.mockCtrl)
	s.manager = managerMocks.NewMockManager(s.mockCtrl)
}

func configWith(v bool) *storage.SensorUpgradeConfig {
	return &storage.SensorUpgradeConfig{EnableAutoUpgrade: v}
}

func (s *SensorUpgradeServiceTestSuite) Test_UpdateSensorUpgradeConfig() {
	testCases := map[string]struct {
		req               *v1.UpdateSensorUpgradeConfigRequest
		managedCentral    bool
		expectedErr       error
		upsertTimesCalled int
	}{
		"Error: No config": {
			req:               &v1.UpdateSensorUpgradeConfigRequest{Config: nil},
			expectedErr:       errox.InvalidArgs,
			upsertTimesCalled: 0,
		},
		"Error: can't set toggle = true on managed centrals": {
			managedCentral: true,
			req: &v1.UpdateSensorUpgradeConfigRequest{
				Config: configWith(true),
			},
			expectedErr:       errox.InvalidArgs,
			upsertTimesCalled: 0,
		},
		"Success: can set toggle = false on managed centrals": {
			managedCentral: true,
			req: &v1.UpdateSensorUpgradeConfigRequest{
				Config: configWith(false),
			},
			upsertTimesCalled: 1,
		},
		"Success: can set toggle = true on non-managed centrals": {
			managedCentral: false,
			req: &v1.UpdateSensorUpgradeConfigRequest{
				Config: configWith(true),
			},
			upsertTimesCalled: 1,
		},
		"Success: can set toggle = false on non-managed centrals": {
			managedCentral: false,
			req: &v1.UpdateSensorUpgradeConfigRequest{
				Config: configWith(false),
			},
			upsertTimesCalled: 1,
		},
	}

	for caseName, testCase := range testCases {
		s.Run(caseName, func() {
			s.T().Setenv(env.ManagedCentral.EnvVar(), strconv.FormatBool(testCase.managedCentral))
			s.dataStore.EXPECT().GetSensorUpgradeConfig(gomock.Any()).Times(1).Return(nil, nil)
			s.dataStore.EXPECT().UpsertSensorUpgradeConfig(gomock.Any(), gomock.Any()).Times(1)
			serviceInstance, err := New(s.dataStore, s.manager)
			s.NoError(err)

			s.dataStore.EXPECT().UpsertSensorUpgradeConfig(gomock.Any(), gomock.Eq(testCase.req.GetConfig())).
				Times(testCase.upsertTimesCalled)
			_, err = serviceInstance.UpdateSensorUpgradeConfig(context.Background(), testCase.req)
			if testCase.expectedErr != nil {
				s.ErrorIs(err, testCase.expectedErr)
			} else {
				s.NoError(err)
			}

			if testCase.upsertTimesCalled > 0 {
				s.Assert().Equal(serviceInstance.AutoUpgradeSetting().Get(),
					testCase.req.GetConfig().GetEnableAutoUpgrade())
			}
		})
	}
}

func (s *SensorUpgradeServiceTestSuite) Test_GetSensorUpgradeConfig_WithValueAlreadyPersisted() {
	s.T().Setenv(env.ManagedCentral.EnvVar(), "false")
	for _, flag := range []bool{true, false} {
		s.Run(fmt.Sprintf("GetSensorUpgradeConfig with value preset = %s", strconv.FormatBool(flag)), func() {
			s.dataStore.EXPECT().GetSensorUpgradeConfig(gomock.Any()).
				Times(2).
				Return(configWith(flag), nil)
			instance, err := New(s.dataStore, s.manager)
			s.NoError(err)
			s.Assert().Equal(flag, instance.AutoUpgradeSetting().Get())

			response, err := instance.GetSensorUpgradeConfig(context.Background(), &v1.Empty{})
			s.NoError(err)

			s.Assert().Equal(flag, response.GetConfig().GetEnableAutoUpgrade())
		})
	}

}

func (s *SensorUpgradeServiceTestSuite) Test_GetSensorUpgradeConfig_WithValueNotPersisted() {
	testCases := map[string]struct {
		expectedAutoUpdate     bool
		expectedFeatureEnabled v1.GetSensorUpgradeConfigResponse_SensorAutoUpgradeFeatureStatus
	}{
		"true": {
			expectedAutoUpdate:     false,
			expectedFeatureEnabled: v1.GetSensorUpgradeConfigResponse_NOT_SUPPORTED,
		},
		"false": {
			expectedAutoUpdate:     true,
			expectedFeatureEnabled: v1.GetSensorUpgradeConfigResponse_SUPPORTED,
		},
	}

	for envValue, expectations := range testCases {
		s.Run(fmt.Sprintf("ROX_MANAGED_CENTRAL=%v", envValue), func() {
			s.dataStore.EXPECT().GetSensorUpgradeConfig(gomock.Any()).Times(1).Return(nil, nil)
			s.dataStore.EXPECT().UpsertSensorUpgradeConfig(gomock.Any(), &UpgradeConfigMatcher{expectations.expectedAutoUpdate})
			s.T().Setenv(env.ManagedCentral.EnvVar(), envValue)

			instance, err := New(s.dataStore, s.manager)
			s.NoError(err)
			s.Assert().Equal(expectations.expectedAutoUpdate, instance.AutoUpgradeSetting().Get())

			s.dataStore.EXPECT().GetSensorUpgradeConfig(gomock.Any()).Times(1).Return(configWith(expectations.expectedAutoUpdate), nil)

			result, err := instance.GetSensorUpgradeConfig(context.Background(), nil)

			s.Require().NoError(err)
			s.Assert().Equal(expectations.expectedAutoUpdate, result.GetConfig().GetEnableAutoUpgrade())
			s.Assert().Equal(expectations.expectedFeatureEnabled, result.GetConfig().GetAutoUpgradeFeature())
		})
	}
}

type UpgradeConfigMatcher struct {
	autoUpgrade bool
}

func (m *UpgradeConfigMatcher) Matches(x interface{}) bool {
	cfg, ok := x.(*storage.SensorUpgradeConfig)
	if !ok {
		return false
	}
	return cfg.EnableAutoUpgrade == m.autoUpgrade
}

func (m *UpgradeConfigMatcher) String() string {
	return fmt.Sprintf("auto-upgrade enabled: %v", m.autoUpgrade)
}

func (s *SensorUpgradeServiceTestSuite) TestAuthzWorks() {
	s.dataStore.EXPECT().GetSensorUpgradeConfig(gomock.Any()).Times(1).Return(configWith(true), nil)
	serviceInstance, err := New(s.dataStore, s.manager)
	s.NoError(err)
	testutils.AssertAuthzWorks(s.T(), serviceInstance)
}
