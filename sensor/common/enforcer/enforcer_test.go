package enforcer

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/protoassert"
	"google.golang.org/protobuf/proto"
)

type alertEnforcementPair struct {
	alert       *storage.Alert
	enforcement storage.EnforcementAction
}

func TestProcessAlertResults(t *testing.T) {
	alert1 := fixtures.GetAlert()
	alert1.GetDeployment().SetId("dep1")
	pi := &storage.ProcessIndicator{}
	pi.SetId("pid")
	pi.SetPodId("pod1")
	ap := &storage.Alert_ProcessViolation{}
	ap.SetProcesses([]*storage.ProcessIndicator{
		pi,
	})
	alert1.SetProcessViolation(ap)

	alert2 := fixtures.GetAlert()
	alert2.GetDeployment().SetId("dep2")

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
				central.SensorEnforcement_builder{
					Enforcement: storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
					Deployment:  proto.ValueOrDefault(generateDeploymentEnforcement(alert1)),
				}.Build(),
				central.SensorEnforcement_builder{
					Enforcement: storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT,
					Deployment:  proto.ValueOrDefault(generateDeploymentEnforcement(alert2)),
				}.Build(),
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
				central.SensorEnforcement_builder{
					Enforcement: storage.EnforcementAction_KILL_POD_ENFORCEMENT,
					ContainerInstance: central.ContainerInstanceEnforcement_builder{
						PodId:                 alert1.GetProcessViolation().GetProcesses()[0].GetPodId(),
						DeploymentEnforcement: generateDeploymentEnforcement(alert1),
					}.Build(),
				}.Build(),
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			enforcer := CreateEnforcer(nil).(*enforcer)

			results := &central.AlertResults{}
			for _, pair := range c.alerts {
				ae := &storage.Alert_Enforcement{}
				ae.SetAction(pair.enforcement)
				pair.alert.SetEnforcement(ae)
				results.SetAlerts(append(results.GetAlerts(), pair.alert))
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
			protoassert.SlicesEqual(t, c.expectedEnforcements, foundEnforcements)
		})
	}
}
