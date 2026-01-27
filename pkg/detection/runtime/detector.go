package runtime

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/kubernetes"
)

// Detector provides an interface for getting and managing alerts and enforcements on deployments.
type Detector interface {
	PolicySet() detection.PolicySet

	DetectForDeploymentAndProcess(enhancedDeployment booleanpolicy.EnhancedDeployment, process *storage.ProcessIndicator, processNotInBaseline bool) ([]*storage.Alert, error)
	DetectForDeploymentAndKubeEvent(enhancedDeployment booleanpolicy.EnhancedDeployment, kubeEvent *storage.KubernetesEvent) ([]*storage.Alert, error)
	DetectForDeploymentAndNetworkFlow(enhancedDeployment booleanpolicy.EnhancedDeployment, flow *augmentedobjs.NetworkFlowDetails) ([]*storage.Alert, error)
	DetectForAuditEvents(auditEvents []*storage.KubernetesEvent) ([]*storage.Alert, error)
	DetectForNodeAndFileAccess(node *storage.Node, access *storage.FileAccess) ([]*storage.Alert, error)
	DetectForDeploymentAndFileAccess(enhancedDeployment booleanpolicy.EnhancedDeployment, access *storage.FileAccess) ([]*storage.Alert, error)
}

// NewDetector returns a new instance of a Detector.
func NewDetector(policySet detection.PolicySet) Detector {
	return &detectorImpl{
		policySet: policySet,
	}
}

type detectorImpl struct {
	policySet detection.PolicySet
}

// PolicySet returns set of policies.
func (d *detectorImpl) PolicySet() detection.PolicySet {
	return d.policySet
}

func (d *detectorImpl) DetectForAuditEvents(auditEvents []*storage.KubernetesEvent) ([]*storage.Alert, error) {
	alerts := make([]*storage.Alert, 0)
	for _, auditEvent := range auditEvents {
		alert, err := d.detectForAuditEvent(auditEvent)
		if err != nil {
			return nil, errors.Wrap(err, "detection on audit events failed")
		}
		alerts = append(alerts, alert...)
	}
	return alerts, nil
}

func (d *detectorImpl) DetectForDeploymentAndProcess(
	enhancedDeployment booleanpolicy.EnhancedDeployment,
	process *storage.ProcessIndicator,
	processNotInBaseline bool,
) ([]*storage.Alert, error) {
	return d.detectForDeployment(enhancedDeployment, process, processNotInBaseline, nil, nil, nil)
}

func (d *detectorImpl) DetectForDeploymentAndKubeEvent(
	enhancedDeployment booleanpolicy.EnhancedDeployment,
	kubeEvent *storage.KubernetesEvent,
) ([]*storage.Alert, error) {
	return d.detectForDeployment(enhancedDeployment, nil, false, kubeEvent, nil, nil)
}

func (d *detectorImpl) DetectForDeploymentAndNetworkFlow(
	enhancedDeployment booleanpolicy.EnhancedDeployment,
	flow *augmentedobjs.NetworkFlowDetails,
) ([]*storage.Alert, error) {
	return d.detectForDeployment(enhancedDeployment, nil, false, nil, flow, nil)
}

func (d *detectorImpl) DetectForDeploymentAndFileAccess(
	enhancedDeployment booleanpolicy.EnhancedDeployment,
	fileAccess *storage.FileAccess,
) ([]*storage.Alert, error) {
	return d.detectForDeployment(enhancedDeployment, nil, false, nil, nil, fileAccess)
}

func (d *detectorImpl) DetectForNodeAndFileAccess(node *storage.Node, access *storage.FileAccess) ([]*storage.Alert, error) {
	var alerts []*storage.Alert
	var cacheReceptacle booleanpolicy.CacheReceptacle

	err := d.policySet.ForEach(func(compiled detection.CompiledPolicy) error {
		if compiled.Policy().GetDisabled() {
			return nil
		}

		if access != nil {
			// Check predicate on file access.
			if !compiled.AppliesTo(access, nil, nil) {
				return nil
			}

			violation, err := compiled.MatchAgainstNodeAndFileAccess(&cacheReceptacle, node, access)
			if err != nil {
				return errors.Wrapf(err, "evaluating violations for policy %q; node file access.",
					compiled.Policy().GetName())
			}

			alert := constructFileAccessAlert(compiled.Policy(), node, nil, violation)
			if alert != nil {
				alerts = append(alerts, alert)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return alerts, nil
}

// detectForDeployment runs detection on a deployment, returning any generated alerts.
func (d *detectorImpl) detectForDeployment(
	enhancedDeployment booleanpolicy.EnhancedDeployment,
	process *storage.ProcessIndicator,
	processNotInBaseline bool,
	kubeEvent *storage.KubernetesEvent,
	flow *augmentedobjs.NetworkFlowDetails,
	fileAccess *storage.FileAccess,
) ([]*storage.Alert, error) {
	var alerts []*storage.Alert
	var cacheReceptable booleanpolicy.CacheReceptacle
	deployment := enhancedDeployment.Deployment

	augmentedDeploy, err := augmentedobjs.ConstructDeployment(deployment, enhancedDeployment.Images, enhancedDeployment.NetworkPoliciesApplied)
	if err != nil {
		return nil, err
	}

	err = d.policySet.ForEach(func(compiled detection.CompiledPolicy) error {
		if compiled.Policy().GetDisabled() {
			return nil
		}

		// Check predicate on deployment.
		if !compiled.AppliesTo(deployment, nil, nil) {
			return nil
		}

		if process != nil {

			violation, err := compiled.MatchAgainstDeploymentAndProcess(&cacheReceptable, enhancedDeployment, process, processNotInBaseline)
			if err != nil {
				return errors.Wrapf(err, "evaluating violations for policy %q; deployment %s/%s",
					compiled.Policy().GetName(), deployment.GetNamespace(), deployment.GetName())
			}

			if alert := constructProcessAlert(compiled.Policy(), deployment, violation); alert != nil {
				alerts = append(alerts, alert)
			}
		}

		if kubeEvent != nil {
			violation, err := compiled.MatchAgainstKubeResourceAndEvent(&cacheReceptable, kubeEvent, augmentedDeploy)
			if err != nil {
				return errors.Wrapf(err, "evaluating violations for policy %q; kubernetes request %s",
					compiled.Policy().GetName(), kubernetes.EventAsString(kubeEvent))
			}

			if alert := constructKubeEventAlert(compiled.Policy(), kubeEvent, deployment, violation); alert != nil {
				alerts = append(alerts, alert)
			}
		}

		if flow != nil {
			violation, err := compiled.MatchAgainstDeploymentAndNetworkFlow(&cacheReceptable, enhancedDeployment, flow)
			if err != nil {
				return errors.Wrapf(err, "evaluating violations for policy %q; network flow %+v",
					compiled.Policy().GetName(), flow)
			}

			if alert := constructNetworkFlowAlert(compiled.Policy(), deployment, flow, violation); alert != nil {
				alerts = append(alerts, alert)
			}
		}

		if fileAccess != nil {
			violation, err := compiled.MatchAgainstDeploymentAndFileAccess(&cacheReceptable, enhancedDeployment, fileAccess)
			if err != nil {
				return errors.Wrapf(err, "evaluating violations for policy %q; file access %+v",
					compiled.Policy().GetName(), fileAccess)
			}

			if alert := constructFileAccessAlert(compiled.Policy(), nil, deployment, violation); alert != nil {
				alerts = append(alerts, alert)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return alerts, nil
}

// detectForAuditEvent runs detection on an audit log event, returning any generated alerts.
func (d *detectorImpl) detectForAuditEvent(auditEvent *storage.KubernetesEvent) ([]*storage.Alert, error) {
	var alerts []*storage.Alert
	var cacheReceptable booleanpolicy.CacheReceptacle

	err := d.policySet.ForEach(func(compiled detection.CompiledPolicy) error {
		if compiled.Policy().GetDisabled() {
			return nil
		}

		if auditEvent != nil {
			// Check predicate on audit event.
			if !compiled.AppliesTo(auditEvent, nil, nil) {
				return nil
			}

			violation, err := compiled.MatchAgainstAuditLogEvent(&cacheReceptable, auditEvent)
			if err != nil {
				return errors.Wrapf(err, "evaluating violations for policy %q; audit log event %s",
					compiled.Policy().GetName(), kubernetes.EventAsString(auditEvent))
			}
			if alert := constructKubeEventAlert(compiled.Policy(), auditEvent, nil, violation); alert != nil {
				alerts = append(alerts, alert)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return alerts, nil
}
