package service

import (
	buildTimeDetection "github.com/stackrox/rox/central/detection/buildtime"
	"github.com/stackrox/rox/central/detection/deploytime"
	"github.com/stackrox/rox/central/enrichment"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(enrichment.ImageEnricherSingleton(),
		enrichment.Singleton(),
		buildTimeDetection.SingletonDetector(),
		deploytime.SingletonDetector(),
		deploytime.SingletonPolicySet())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
