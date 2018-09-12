package pipeline

import "github.com/stackrox/rox/generated/api/v1"

// Pipeline represents the processing applied to a SensorEvent to produce a response.
type Pipeline interface {
	Run(event *v1.SensorEvent) (*v1.SensorEnforcement, error)
}
