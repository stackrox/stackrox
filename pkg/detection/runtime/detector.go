package runtime

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Detector provides an interface for getting and managing alerts and enforcements on deployments.
type Detector interface {
	PolicySet() detection.PolicySet

	Detect(deployment *storage.Deployment, images []*storage.Image, process *storage.ProcessIndicator, processOutsideWhitelist bool) ([]*storage.Alert, error)
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

// Detect runs detection on an deployment, returning any generated alerts.
func (d *detectorImpl) Detect(deployment *storage.Deployment, images []*storage.Image, indicator *storage.ProcessIndicator, processOutsideWhitelist bool) ([]*storage.Alert, error) {
	var alerts []*storage.Alert
	err := d.policySet.ForEach(func(compiled detection.CompiledPolicy) error {
		if compiled.Policy().GetDisabled() {
			return nil
		}
		// Check predicate on deployment.
		if !compiled.AppliesTo(deployment) {
			return nil
		}

		violation, err := compiled.MatchAgainstDeploymentAndProcess(deployment, images, indicator, processOutsideWhitelist)
		if err != nil {
			return errors.Wrapf(err, "evaluating violations for policy %s; deployment %s/%s", compiled.Policy().GetName(), deployment.GetNamespace(), deployment.GetName())
		}

		if alert := policyDeploymentAndViolationsToAlert(compiled.Policy(), deployment, violation); alert != nil {
			alerts = append(alerts, alert)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return alerts, nil
}
