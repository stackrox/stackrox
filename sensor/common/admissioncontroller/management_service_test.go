package admissioncontroller

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	alertHandlerMocks "github.com/stackrox/rox/sensor/common/admissioncontroller/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestManagementService(t *testing.T) {
	suite.Run(t, new(managementServiceSuite))
}

type managementServiceSuite struct {
	suite.Suite
	mockCtrl            *gomock.Controller
	mockAlertHandler    *alertHandlerMocks.MockAlertHandler
	mockSettingsManager *alertHandlerMocks.MockSettingsManager
	service             *managementService
}

var _ suite.SetupTestSuite = (*managementServiceSuite)(nil)

func (s *managementServiceSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockAlertHandler = alertHandlerMocks.NewMockAlertHandler(s.mockCtrl)
	s.mockSettingsManager = alertHandlerMocks.NewMockSettingsManager(s.mockCtrl)
}

func (s *managementServiceSuite) createManagementService() {
	s.mockSettingsManager.EXPECT().SettingsStream().Times(1)
	s.mockSettingsManager.EXPECT().SensorEventsStream().Times(1)
	s.service = &managementService{
		settingsStream:     s.mockSettingsManager.SettingsStream(),
		sensorEventsStream: s.mockSettingsManager.SensorEventsStream(),
		alertHandler:       s.mockAlertHandler,
		admCtrlMgr:         s.mockSettingsManager,
	}
}

func (s *managementServiceSuite) TestPolicyAlerts() {
	ctx := context.Background()
	cases := map[string]struct {
		request             *sensor.AdmissionControlAlerts
		expectProcessAlerts func()
		err                 error
	}{
		"AlertHandler returns an error": {
			request:             createAlertRequest("1", "1", storage.LifecycleStage_DEPLOY, central.AlertResults_DEPLOYMENT_EVENT),
			expectProcessAlerts: expectProcessAlerts(s.mockAlertHandler, 1, errCentralNoReachable),
			err:                 errCentralNoReachable,
		},
		"AlertHandler successes": {
			request:             createAlertRequest("1", "1", storage.LifecycleStage_DEPLOY, central.AlertResults_DEPLOYMENT_EVENT),
			expectProcessAlerts: expectProcessAlerts(s.mockAlertHandler, 1, nil),
			err:                 nil,
		},
	}
	for testName, c := range cases {
		s.Run(testName, func() {
			s.createManagementService()
			c.expectProcessAlerts()
			_, err := s.service.PolicyAlerts(ctx, c.request)
			if c.err != nil {
				s.Assert().EqualError(err, errCentralNoReachable.Error())
			} else {
				s.Assert().NoError(err)
			}
		})
	}
}

func expectProcessAlerts(mockAlertHandler *alertHandlerMocks.MockAlertHandler, times int, retErr error) func() {
	return func() {
		mockAlertHandler.EXPECT().ProcessAlerts(gomock.Any()).Times(times).DoAndReturn(func(_ any) error {
			return retErr
		})
	}
}

func createAlertRequest(deploymentID, policyID string, stage storage.LifecycleStage, source central.AlertResults_Source) *sensor.AdmissionControlAlerts {
	return &sensor.AdmissionControlAlerts{
		AlertResults: []*central.AlertResults{
			{
				DeploymentId: deploymentID,
				Alerts: []*storage.Alert{
					{
						Policy: &storage.Policy{
							Id: policyID,
						},
					},
				},
				Stage:  stage,
				Source: source,
			},
		},
	}
}
