package runtime

import (
	"fmt"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection"
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/uuid"
)

type alertCollectingExecutor interface {
	detection.PolicyExecutor

	GetAlerts() []*storage.Alert
	ClearAlerts()
}

func newAlertCollectingExecutor(deployments datastore.DataStore, deploymentIDs ...string) alertCollectingExecutor {
	return &alertCollectingExecutorImpl{
		deploymentIDs: deploymentIDs,
		deployments:   deployments,
	}
}

type alertCollectingExecutorImpl struct {
	deploymentIDs []string
	deployments   datastore.DataStore
	alerts        []*storage.Alert
}

func (d *alertCollectingExecutorImpl) GetAlerts() []*storage.Alert {
	return d.alerts
}

func (d *alertCollectingExecutorImpl) ClearAlerts() {
	d.alerts = nil
}

// IsProcessWhitelistPolicy returns if the whitelist enabled field is set to true
func IsProcessWhitelistPolicy(compiled detection.CompiledPolicy) bool {
	return compiled.Policy().GetFields().GetWhitelistEnabled()
}

func (d *alertCollectingExecutorImpl) Execute(compiled detection.CompiledPolicy) error {
	if IsProcessWhitelistPolicy(compiled) {
		return nil
	}
	var err error
	var violationsByDeployment map[string]searchbasedpolicies.Violations
	if len(d.deploymentIDs) == 0 {
		violationsByDeployment, err = compiled.Matcher().Match(d.deployments)
	} else {
		violationsByDeployment, err = compiled.Matcher().MatchMany(d.deployments, d.deploymentIDs...)
	}

	if err != nil {
		return errors.Wrapf(err, "matching policy %s", compiled.Policy().GetName())
	}

	for deploymentID, violations := range violationsByDeployment {
		dep, exists, err := d.deployments.GetDeployment(deploymentID)
		if err != nil {
			return err
		}
		if !exists {
			log.Errorf("deployment with id '%s' had violations, but doesn't exist", deploymentID)
			continue
		}
		if !compiled.AppliesTo(dep) {
			continue
		}
		d.alerts = append(d.alerts, PolicyDeploymentAndViolationsToAlert(compiled.Policy(), dep, violations))
	}
	return nil
}

// PolicyDeploymentAndViolationsToAlert constructs an alert.
func PolicyDeploymentAndViolationsToAlert(policy *storage.Policy, deployment *storage.Deployment, violations searchbasedpolicies.Violations) *storage.Alert {
	if len(violations.AlertViolations) == 0 && violations.ProcessViolation == nil {
		return nil
	}
	alert := &storage.Alert{
		Id:               uuid.NewV4().String(),
		LifecycleStage:   storage.LifecycleStage_RUNTIME,
		Deployment:       protoutils.CloneStorageDeployment(deployment),
		Policy:           protoutils.CloneStoragePolicy(policy),
		Violations:       violations.AlertViolations,
		ProcessViolation: violations.ProcessViolation,
		Time:             ptypes.TimestampNow(),
	}
	if action, msg := buildEnforcement(policy, deployment); action != storage.EnforcementAction_UNSET_ENFORCEMENT {
		alert.Enforcement = &storage.Alert_Enforcement{
			Action:  action,
			Message: msg,
		}
	}
	return alert
}

func buildEnforcement(policy *storage.Policy, deployment *storage.Deployment) (enforcement storage.EnforcementAction, message string) {
	for _, enforcementAction := range policy.GetEnforcementActions() {
		if enforcementAction == storage.EnforcementAction_KILL_POD_ENFORCEMENT {
			return storage.EnforcementAction_KILL_POD_ENFORCEMENT, fmt.Sprintf("Deployment %s has pods killed in response to policy violation", deployment.GetName())
		}
	}
	return storage.EnforcementAction_UNSET_ENFORCEMENT, ""
}
