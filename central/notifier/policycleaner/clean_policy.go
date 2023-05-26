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
	err := s.buildTimePolicies.RemoveNotifier(notifierID)
	if err != nil {
		return err
	}

	err = s.deployTimePolicies.RemoveNotifier(notifierID)
	if err != nil {
		return err
	}

	err = s.runTimePolicies.RemoveNotifier(notifierID)
	if err != nil {
		return err
	}

	return nil
}
