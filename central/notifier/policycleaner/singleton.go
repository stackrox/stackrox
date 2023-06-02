package policycleaner

import (
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
