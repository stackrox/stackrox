package service

import (
	buildTimeDetection "github.com/stackrox/rox/central/detection/buildtime"
	deployTimeDetection "github.com/stackrox/rox/central/detection/deploytime"
	runTimeDetection "github.com/stackrox/rox/central/detection/runtime"
	healthDatastore "github.com/stackrox/rox/central/integrationhealth/datastore"
	"github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/notifier/processor"
	"github.com/stackrox/rox/pkg/sync"
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
		healthDatastore.Singleton(),
	)
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
