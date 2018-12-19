package pipeline

import (
	"github.com/stackrox/rox/generated/internalapi/central"
)

// Pipeline represents the processing applied to a SensorEvent to produce a response.
//go:generate mockgen-wrapper Pipeline
type Pipeline interface {
	Run(event *central.SensorEvent, injector EnforcementInjector) error
}

// An EnforcementInjector allows a pipeline to return an enforcement action back into the pipeline.
// It does a best-effort send, and returns a bool whether it succeeded. (It will fail if the stream in the
// time between the object being passed to the pipeline and the stream being broken.)
type EnforcementInjector interface {
	InjectEnforcement(*central.SensorEnforcement) bool
}
