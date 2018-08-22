package service

import (
	"sync"

	buildTimeDetection "github.com/stackrox/rox/central/detection/buildtime"
	runTimeDetection "github.com/stackrox/rox/central/detection/runtime"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(buildTimeDetection.SingletonDetector(), runTimeDetection.SingletonDetector())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
