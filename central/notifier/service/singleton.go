package service

import (
	buildTimeDetection "github.com/stackrox/stackrox/central/detection/buildtime"
	deployTimeDetection "github.com/stackrox/stackrox/central/detection/deploytime"
	runTimeDetection "github.com/stackrox/stackrox/central/detection/runtime"
	"github.com/stackrox/stackrox/central/integrationhealth/reporter"
	"github.com/stackrox/stackrox/central/notifier/datastore"
	"github.com/stackrox/stackrox/central/notifier/processor"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(
		datastore.Singleton(),
		processor.Singleton(),
		buildTimeDetection.SingletonPolicySet(),
		deployTimeDetection.SingletonPolicySet(),
		runTimeDetection.SingletonPolicySet(),
		reporter.Singleton(),
	)
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
