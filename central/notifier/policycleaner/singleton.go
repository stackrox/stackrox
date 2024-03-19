package policycleaner

import (
	"testing"

	"github.com/stackrox/rox/central/detection"
	buildTimeDetection "github.com/stackrox/rox/central/detection/buildtime"
	deployTimeDetection "github.com/stackrox/rox/central/detection/deploytime"
	runTimeDetection "github.com/stackrox/rox/central/detection/runtime"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	pc PolicyCleaner
)

func initialize() {
	pc = PolicyCleaner{
		buildTimePolicies:  buildTimeDetection.SingletonPolicySet(),
		deployTimePolicies: deployTimeDetection.SingletonPolicySet(),
		runTimePolicies:    runTimeDetection.SingletonPolicySet(),
	}
}

// Singleton provides the instance of the Service interface to register.
func Singleton() PolicyCleaner {
	once.Do(initialize)
	return pc
}

// GetTestPolicyCleaner provides a policy cleaner that can be used for testing purposes.
func GetTestPolicyCleaner(_ *testing.T, set detection.PolicySet) PolicyCleaner {
	return PolicyCleaner{
		buildTimePolicies:  set,
		deployTimePolicies: set,
		runTimePolicies:    set,
	}
}
