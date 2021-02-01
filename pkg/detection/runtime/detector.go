package runtime

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Detector provides an interface for getting and managing alerts and enforcements on deployments.
type Detector interface {
	PolicySet() detection.PolicySet

	DetectForDeploymentAndProcess(deployment *storage.Deployment, images []*storage.Image, process *storage.ProcessIndicator, processNotInBaseline bool) ([]*storage.Alert, error)
	DetectForDeploymentAndKubeEvent(deployment *storage.Deployment, images []*storage.Image, kubeEvent *storage.KubernetesEvent) ([]*storage.Alert, error)
	DetectForDeploymentAndNetworkFlow(deployment *storage.Deployment, images []*storage.Image, flow *augmentedobjs.NetworkFlowDetails) ([]*storage.Alert, error)
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

// UpsertPolicy adds or updates a policy in the set.
func (d *detectorImpl) PolicySet() detection.PolicySet {
	return d.policySet
}

func (d *detectorImpl) DetectForDeploymentAndProcess(
	deployment *storage.Deployment,
	images []*storage.Image,
	process *storage.ProcessIndicator,
	processNotInBaseline bool,
) ([]*storage.Alert, error) {
	return d.detectForDeployment(deployment, images, process, processNotInBaseline, nil, nil)
}

func (d *detectorImpl) DetectForDeploymentAndKubeEvent(
	deployment *storage.Deployment,
	images []*storage.Image,
	kubeEvent *storage.KubernetesEvent,
) ([]*storage.Alert, error) {
	return d.detectForDeployment(deployment, images, nil, false, kubeEvent, nil)
}

func (d *detectorImpl) DetectForDeploymentAndNetworkFlow(
	deployment *storage.Deployment,
	images []*storage.Image,
	flow *augmentedobjs.NetworkFlowDetails,
) ([]*storage.Alert, error) {
	return d.detectForDeployment(deployment, images, nil, false, nil, flow)
}

// detectForDeployment runs detection on a deployment, returning any generated alerts.
func (d *detectorImpl) detectForDeployment(
	deployment *storage.Deployment,
	images []*storage.Image,
	process *storage.ProcessIndicator,
	processNotInBaseline bool,
	kubeEvent *storage.KubernetesEvent,
	flow *augmentedobjs.NetworkFlowDetails,
) ([]*storage.Alert, error) {
	var alerts []*storage.Alert
	var cacheReceptable booleanpolicy.CacheReceptacle

	augmentedDeploy, err := augmentedobjs.ConstructDeployment(deployment, images)
	if err != nil {
		return nil, err
	}

	err = d.policySet.ForEach(func(compiled detection.CompiledPolicy) error {
		if compiled.Policy().GetDisabled() {
			return nil
		}

		// Check predicate on deployment.
		if !compiled.AppliesTo(deployment) {
			return nil
		}

		if process != nil {
			violation, err := compiled.MatchAgainstDeploymentAndProcess(&cacheReceptable, deployment, images, process, processNotInBaseline)
			if err != nil {
				return errors.Wrapf(err, "evaluating violations for policy %q; deployment %s/%s",
					compiled.Policy().GetName(), deployment.GetNamespace(), deployment.GetName())
			}

			if alert := constructProcessAlert(compiled.Policy(), deployment, violation); alert != nil {
				alerts = append(alerts, alert)
			}
		}

		if kubeEvent != nil && features.K8sEventDetection.Enabled() {
			violation, err := compiled.MatchAgainstKubeResourceAndEvent(&cacheReceptable, kubeEvent, augmentedDeploy)
			if err != nil {
				return errors.Wrapf(err, "evaluating violations for policy %q; kubernetes request %s",
					compiled.Policy().GetName(), kubernetes.EventAsString(kubeEvent))
			}

			if alert := constructKubeEventAlert(compiled.Policy(), kubeEvent, deployment, violation); alert != nil {
				alerts = append(alerts, alert)
			}
		}

		if flow != nil && features.NetworkDetectionBaselineViolation.Enabled() {
			violation, err := compiled.MatchAgainstDeploymentAndNetworkFlow(&cacheReceptable, deployment, images, flow)
			if err != nil {
				return errors.Wrapf(err, "evaluating violations for policy %q; network flow %+v",
					compiled.Policy().GetName(), flow)
			}

			if alert := constructNetworkFlowAlert(compiled.Policy(), deployment, flow, violation); alert != nil {
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
