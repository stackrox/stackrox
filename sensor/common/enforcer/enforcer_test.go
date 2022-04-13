package enforcer

import (
	"testing"

	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
)

type alertEnforcementPair struct {
	alert       *storage.Alert
	enforcement storage.EnforcementAction
}

func TestProcessAlertResults(t *testing.T) {
	alert1 := fixtures.GetAlert()
	alert1.GetDeployment().Id = "dep1"
	alert1.ProcessViolation = &storage.Alert_ProcessViolation{
		Processes: []*storage.ProcessIndicator{
			{
				Id:    "pid",
				PodId: "pod1",
			},
		},
	}

	alert2 := fixtures.GetAlert()
	alert2.GetDeployment().Id = "dep2"

	cases := []struct {
		name                 string
		action               central.ResourceAction
		stage                storage.LifecycleStage
		alerts               []alertEnforcementPair
		expectedEnforcements []*central.SensorEnforcement
	}{
		{
			name:   "update action - no output",
			action: central.ResourceAction_UPDATE_RESOURCE,
			stage:  storage.LifecycleStage_DEPLOY,
		},
		{
			name:   "remove action - no output",
			action: central.ResourceAction_REMOVE_RESOURCE,
			stage:  storage.LifecycleStage_DEPLOY,
		},
		{
			name:   "unset action - no output",
			action: central.ResourceAction_UNSET_ACTION_RESOURCE,
			stage:  storage.LifecycleStage_DEPLOY,
		},
		{
			name:   "create action - all alerts have unset enforcement",
			action: central.ResourceAction_CREATE_RESOURCE,
			stage:  storage.LifecycleStage_DEPLOY,
			alerts: []alertEnforcementPair{
				{
					alert:       alert1,
					enforcement: storage.EnforcementAction_UNSET_ENFORCEMENT,
				},
				{
					alert:       alert2,
					enforcement: storage.EnforcementAction_UNSET_ENFORCEMENT,
				},
			},
		},
		{
			name:   "create action - 2 alerts are enforced",
			action: central.ResourceAction_CREATE_RESOURCE,
			stage:  storage.LifecycleStage_DEPLOY,
			alerts: []alertEnforcementPair{
				{
					alert:       alert1,
					enforcement: storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
				},
				{
					alert:       alert2,
					enforcement: storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT,
				},
			},
			expectedEnforcements: []*central.SensorEnforcement{
				{
					Enforcement: storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
					Resource: &central.SensorEnforcement_Deployment{
						Deployment: generateDeploymentEnforcement(alert1),
					},
				},
				{
					Enforcement: storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT,
					Resource: &central.SensorEnforcement_Deployment{
						Deployment: generateDeploymentEnforcement(alert2),
					},
				},
			},
		},
		{
			name:   "update action - 1 alert is enforced, no enforcement",
			action: central.ResourceAction_UPDATE_RESOURCE,
			stage:  storage.LifecycleStage_DEPLOY,
			alerts: []alertEnforcementPair{
				{
					alert:       alert1,
					enforcement: storage.EnforcementAction_KILL_POD_ENFORCEMENT,
				},
			},
		},
		{
			name:   "create action, runtime lifecycle - 1 alert is enforced",
			action: central.ResourceAction_CREATE_RESOURCE,
			stage:  storage.LifecycleStage_RUNTIME,
			alerts: []alertEnforcementPair{
				{
					alert:       alert1,
					enforcement: storage.EnforcementAction_KILL_POD_ENFORCEMENT,
				},
			},
			expectedEnforcements: []*central.SensorEnforcement{
				{
					Enforcement: storage.EnforcementAction_KILL_POD_ENFORCEMENT,
					Resource: &central.SensorEnforcement_ContainerInstance{
						ContainerInstance: &central.ContainerInstanceEnforcement{
							PodId:                 alert1.GetProcessViolation().GetProcesses()[0].GetPodId(),
							DeploymentEnforcement: generateDeploymentEnforcement(alert1),
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			enforcer := CreateEnforcer(nil).(*enforcer)

			results := &central.AlertResults{}
			for _, pair := range c.alerts {
				pair.alert.Enforcement = &storage.Alert_Enforcement{
					Action: pair.enforcement,
				}
				results.Alerts = append(results.Alerts, pair.alert)
			}

			enforcer.ProcessAlertResults(c.action, c.stage, results)
			var foundEnforcements []*central.SensorEnforcement
		LOOP:
			for {
				select {
				case enforcement := <-enforcer.actionsC:
					foundEnforcements = append(foundEnforcements, enforcement)
				default:
					break LOOP
				}
			}
			assert.Equal(t, c.expectedEnforcements, foundEnforcements)
		})
	}
}
