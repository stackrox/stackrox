package deploytime

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	deploymentIndexer "github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/detection/deployment"
	"github.com/stackrox/rox/central/globalindex"
	imageIndexer "github.com/stackrox/rox/central/image/index"
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compiledpolicies/deployment/predicate"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	deploymentID = "deployment-id"
)

var (
	log = logging.LoggerForModule()
)

type detectorImpl struct {
	policySet deployment.PolicySet

	deployments deploymentDataStore.DataStore
}

func tempDeploymentIndexer(deployment *storage.Deployment) (deploymentIndexer.Indexer, error) {
	clonedDeployment := proto.Clone(deployment).(*storage.Deployment)
	if clonedDeployment.GetId() == "" {
		clonedDeployment.Id = deploymentID
	}
	tempIndex, err := globalindex.MemOnlyIndex()
	if err != nil {
		return nil, fmt.Errorf("initializing temp index: %s", err)
	}
	imageIndex := imageIndexer.New(tempIndex)
	deploymentIndex := deploymentIndexer.New(tempIndex)
	for i, container := range clonedDeployment.GetContainers() {
		if container.GetImage() == nil {
			continue
		}
		if container.GetImage().GetId() == "" {
			container.Image.Id = fmt.Sprintf("image-id-%d", i)
		}
		if err := imageIndex.AddImage(container.GetImage()); err != nil {
			return nil, err
		}
	}
	if err := deploymentIndex.AddDeployment(clonedDeployment); err != nil {
		return nil, err
	}
	return deploymentIndex, nil
}

// Detect runs detection on an deployment, returning any generated alerts.
func (d *detectorImpl) Detect(deployment *storage.Deployment) ([]*storage.Alert, error) {
	deploymentIndex, err := tempDeploymentIndexer(deployment)
	if err != nil {
		return nil, err
	}
	return d.evaluateAlertsForDeployment(deploymentIndex, deployment)
}

// UpsertPolicy adds or updates a policy in the set.
func (d *detectorImpl) UpsertPolicy(policy *storage.Policy) error {
	return d.policySet.UpsertPolicy(policy)
}

// RemovePolicy removes a policy from the set.
func (d *detectorImpl) RemovePolicy(policyID string) error {
	return d.policySet.RemovePolicy(policyID)
}

func (d *detectorImpl) matchWithEmptyImageIDs(p *storage.Policy, matcher searchbasedpolicies.Matcher, deployment *storage.Deployment) ([]*storage.Alert_Violation, error) {
	deploymentIndex, err := tempDeploymentIndexer(deployment)
	violations, err := matcher.MatchOne(deploymentIndex, deployment.GetId())
	if err != nil {
		return nil, err
	}
	return violations.AlertViolations, nil
}

func (d *detectorImpl) evaluateAlertsForDeployment(searcher searchbasedpolicies.Searcher, deployment *storage.Deployment) ([]*storage.Alert, error) {
	var newAlerts []*storage.Alert
	err := d.policySet.ForEach(func(p *storage.Policy, matcher searchbasedpolicies.Matcher, shouldProcess predicate.Predicate) error {
		if shouldProcess != nil && !shouldProcess(deployment) {
			return nil
		}

		var err error
		var violations []*storage.Alert_Violation
		if enforcement, _ := policyAndDeploymentToEnforcement(p, deployment); enforcement != storage.EnforcementAction_UNSET_ENFORCEMENT {
			violations, err = d.matchWithEmptyImageIDs(p, matcher, deployment)
		} else {
			var violationsWrapper searchbasedpolicies.Violations
			// Purposefully, use searcher for deployment check
			violationsWrapper, err = matcher.MatchOne(searcher, deployment.GetId())
			violations = violationsWrapper.AlertViolations
		}
		if err != nil {
			return fmt.Errorf("evaluating violations for policy %s; deployment %s/%s: %s", p.GetName(), deployment.GetNamespace(), deployment.GetName(), err)
		}
		if len(violations) > 0 {
			newAlerts = append(newAlerts, policyDeploymentAndViolationsToAlert(p, deployment, violations))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return newAlerts, nil
}

func (d *detectorImpl) AlertsForDeployment(deployment *storage.Deployment) ([]*storage.Alert, error) {
	return d.evaluateAlertsForDeployment(d.deployments, deployment)
}

func (d *detectorImpl) AlertsForPolicy(policyID string) ([]*storage.Alert, error) {
	var newAlerts []*storage.Alert
	err := d.policySet.ForOne(policyID, func(p *storage.Policy, matcher searchbasedpolicies.Matcher, shouldProcess predicate.Predicate) error {
		violationsByDeployment, err := matcher.Match(d.deployments)
		if err != nil {
			return err
		}
		for deploymentID, violations := range violationsByDeployment {
			dep, exists, err := d.deployments.GetDeployment(deploymentID)
			if err != nil {
				return err
			}
			if !exists {
				log.Errorf("deployment with id %q had violations, but doesn't exist", deploymentID)
				continue
			}
			if shouldProcess != nil && !shouldProcess(dep) {
				continue
			}
			newAlerts = append(newAlerts, policyDeploymentAndViolationsToAlert(p, dep, violations.AlertViolations))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return newAlerts, nil
}

// policyDeploymentAndViolationsToAlert constructs an alert.
func policyDeploymentAndViolationsToAlert(policy *storage.Policy, deployment *storage.Deployment, violations []*storage.Alert_Violation) *storage.Alert {
	if len(violations) == 0 {
		return nil
	}
	alert := &storage.Alert{
		Id:             uuid.NewV4().String(),
		LifecycleStage: storage.LifecycleStage_DEPLOY,
		Deployment:     protoutils.CloneStorageDeployment(deployment),
		Policy:         protoutils.CloneStoragePolicy(policy),
		Violations:     violations,
		Time:           ptypes.TimestampNow(),
	}
	if action, msg := policyAndDeploymentToEnforcement(policy, deployment); action != storage.EnforcementAction_UNSET_ENFORCEMENT {
		alert.Enforcement = &storage.Alert_Enforcement{
			Action:  action,
			Message: msg,
		}
	}
	return alert
}

// policyAndDeploymentToEnforcement returns enforcement info for a deployment violating a policy.
func policyAndDeploymentToEnforcement(policy *storage.Policy, deployment *storage.Deployment) (enforcement storage.EnforcementAction, message string) {
	for _, enforcementAction := range policy.GetEnforcementActions() {
		if enforcementAction == storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT && scaleToZeroEnabled(deployment) {
			return storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT, fmt.Sprintf("Deployment %s scaled to 0 replicas in response to policy violation", deployment.GetName())
		}
		if enforcementAction == storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT {
			return storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT, fmt.Sprintf("Unsatisfiable node constraint applied to deployment %s", deployment.GetName())
		}
	}
	return storage.EnforcementAction_UNSET_ENFORCEMENT, ""
}

const (
	globalDeployment    = "Global"
	daemonSetDeployment = "DaemonSet"
)

func scaleToZeroEnabled(deployment *storage.Deployment) bool {
	if deployment.GetType() == globalDeployment || deployment.GetType() == daemonSetDeployment {
		return false
	}
	return true
}
