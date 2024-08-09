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

var _ suite.SetupSubTest = (*SensorUpgradeServiceTestSuite)(nil)

func (s *SensorUpgradeServiceTestSuite) SetupSubTest() {
	// Each subtest must use its own T object, thus the controller must be reinitialized for each subtest
	s.mockCtrl = gomock.NewController(s.T())
	s.dataStore = datastoreMocks.NewMockDataStore(s.mockCtrl)
	s.manager = managerMocks.NewMockManager(s.mockCtrl)
}

func (s *SensorUpgradeServiceTestSuite) SetupTest() {
	// Not every test has subtest, so the same setup procedure must be repeated for standalone tests
	s.SetupSubTest()
}

func configWith(v bool) *storage.SensorUpgradeConfig {
	return &storage.SensorUpgradeConfig{EnableAutoUpgrade: v}
}

func (s *SensorUpgradeServiceTestSuite) Test_UpdateSensorUpgradeConfig() {
	testCases := map[string]struct {
		req               *v1.UpdateSensorUpgradeConfigRequest
		managedCentral    bool
		upgraderEnabled   bool
		expectedErr       error
		upsertTimesCalled int
	}{
		"Nil config should yield an error": {
			upgraderEnabled:   true,
			req:               &v1.UpdateSensorUpgradeConfigRequest{Config: nil},
			expectedErr:       errox.InvalidArgs,
			upsertTimesCalled: 0,
		},
		"Enabling upgrader through the config should work on managed centrals": {
			managedCentral:  true,
			upgraderEnabled: true,
			req: &v1.UpdateSensorUpgradeConfigRequest{
				Config: configWith(true),
			},
			expectedErr:       nil,
			upsertTimesCalled: 1,
		},
		"Enabling upgrader through the config should fail on managed centrals if upgrader is explicitly disabled": {
			managedCentral:  true,
			upgraderEnabled: false,
			req: &v1.UpdateSensorUpgradeConfigRequest{
				Config: configWith(true),
			},
			expectedErr:       errox.InvalidArgs,
			upsertTimesCalled: 0,
		},
		"Disabling upgrader through the config should work on managed centrals": {
			managedCentral:  true,
			upgraderEnabled: true,
			req: &v1.UpdateSensorUpgradeConfigRequest{
				Config: configWith(false),
			},
			upsertTimesCalled: 1,
		},
		"Enabling upgrader through the config should work on non-CS centrals": {
			managedCentral:  false,
			upgraderEnabled: true,
			req: &v1.UpdateSensorUpgradeConfigRequest{
				Config: configWith(true),
			},
			upsertTimesCalled: 1,
		},
		"Disabling upgrader through the config should work on non-CS centrals": {
			managedCentral:  false,
			upgraderEnabled: true,
			req: &v1.UpdateSensorUpgradeConfigRequest{
				Config: configWith(false),
			},
			upsertTimesCalled: 1,
		},
		"Disabling upgrader through the config should not yield an error on non-CS centrals even if upgrader is explicitly disabled": {
			managedCentral:  false,
			upgraderEnabled: false,
			req: &v1.UpdateSensorUpgradeConfigRequest{
				Config: configWith(false),
			},
			expectedErr:       nil,
			upsertTimesCalled: 1,
		},
	}

	for caseName, testCase := range testCases {
		s.Run(caseName, func() {
			s.T().Setenv(env.ManagedCentral.EnvVar(), strconv.FormatBool(testCase.managedCentral))
			s.T().Setenv(env.SensorUpgraderEnabled.EnvVar(), strconv.FormatBool(testCase.upgraderEnabled))
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
		"false": {
			expectedAutoUpdate:     false,
			expectedFeatureEnabled: v1.GetSensorUpgradeConfigResponse_NOT_SUPPORTED,
		},
		"true": {
			expectedAutoUpdate:     true,
			expectedFeatureEnabled: v1.GetSensorUpgradeConfigResponse_SUPPORTED,
		},
	}

	for envValue, expectations := range testCases {
		s.Run(fmt.Sprintf("%s=%v", env.SensorUpgraderEnabled.EnvVar(), envValue), func() {
			s.dataStore.EXPECT().GetSensorUpgradeConfig(gomock.Any()).Times(1).Return(nil, nil)
			s.dataStore.EXPECT().UpsertSensorUpgradeConfig(gomock.Any(), &UpgradeConfigMatcher{expectations.expectedAutoUpdate})
			s.T().Setenv(env.SensorUpgraderEnabled.EnvVar(), envValue)

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
