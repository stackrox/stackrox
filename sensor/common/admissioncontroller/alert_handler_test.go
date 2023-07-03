package admissioncontroller

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stretchr/testify/suite"
)

const (
	defaultTimeout = 1 * time.Second
)

func TestAlertHandler(t *testing.T) {
	suite.Run(t, new(alertHandlerSuite))
}

type alertHandlerSuite struct {
	suite.Suite
}

func (s *alertHandlerSuite) TestProcessAlert() {
	cases := map[string]struct {
		notify       common.SensorComponentEvent
		deploymentID string
		policyID     string
		stage        storage.LifecycleStage
		source       central.AlertResults_Source
		err          error
	}{
		"Central reachable": {
			notify:       common.SensorComponentEventCentralReachable,
			deploymentID: "1",
			policyID:     "1",
			stage:        storage.LifecycleStage_DEPLOY,
			source:       central.AlertResults_DEPLOYMENT_EVENT,
			err:          nil,
		},
		"Central unreachable": {
			notify:       common.SensorComponentEventOfflineMode,
			deploymentID: "1",
			policyID:     "1",
			stage:        storage.LifecycleStage_DEPLOY,
			source:       central.AlertResults_DEPLOYMENT_EVENT,
			err:          errCentralNoReachable,
		},
	}
	for testName, c := range cases {
		s.Run(testName, func() {
			h := newAlertHandler()
			h.Notify(c.notify)
			admissionControlAlerts := createAlertResults(c.deploymentID, c.policyID, c.stage, c.source)
			err := h.ProcessAlerts(admissionControlAlerts)
			if c.err != nil {
				s.Assert().EqualError(err, errCentralNoReachable.Error())
			} else {
				s.Assert().NoError(err)
				select {
				case <-time.After(defaultTimeout):
					s.T().Error("timeout reached waiting for alert results")
				case res, ok := <-h.ResponsesC():
					if !ok {
						s.T().Error("ResponsesC should not be closed")
					}
					s.Assert().Equal(createAlertResultsMsg(admissionControlAlerts.AlertResults[0]), res)
				}
			}
		})
	}
}

func createAlertResults(deploymentID, policyID string, stage storage.LifecycleStage, source central.AlertResults_Source) *sensor.AdmissionControlAlerts {
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
