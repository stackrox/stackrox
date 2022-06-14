package deploytime

import (
	"github.com/stackrox/stackrox/central/detection"
	policyDataStore "github.com/stackrox/stackrox/central/policy/datastore"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	policySet detection.PolicySet
	detector  Detector
)

// SingletonDetector returns the singleton instance of a Detector.
func SingletonDetector() Detector {
	once.Do(initialize)
	return detector
}

// SingletonPolicySet returns the singleton instance of a PolicySet.
func SingletonPolicySet() detection.PolicySet {
	once.Do(initialize)
	return policySet
}

func initialize() {
	policySet = detection.NewPolicySet(policyDataStore.Singleton())
	detector = NewDetector(policySet)
}
