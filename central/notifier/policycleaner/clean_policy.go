package policycleaner

import "github.com/stackrox/rox/central/detection"

// PolicyCleaner removes notifier from policies.
type PolicyCleaner struct {
	buildTimePolicies  detection.PolicySet
	deployTimePolicies detection.PolicySet
	runTimePolicies    detection.PolicySet
}

// DeleteNotifierFromPolicies removes notifier from policies.
func (p *PolicyCleaner) DeleteNotifierFromPolicies(notifierID string) error {
	if err := p.buildTimePolicies.RemoveNotifier(notifierID); err != nil {
		return err
	}

	if err := p.deployTimePolicies.RemoveNotifier(notifierID); err != nil {
		return err
	}

	if err := p.runTimePolicies.RemoveNotifier(notifierID); err != nil {
		return err
	}

	return nil
}
