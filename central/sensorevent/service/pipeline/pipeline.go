package pipeline

import (
	"github.com/stackrox/rox/generated/api/v1"
)

// Pipeline represents the processing applied to a SensorEvent to produce a response.
//go:generate mockery -name=Pipeline
type Pipeline interface {
	Run(event *v1.SensorEvent) (*v1.SensorEnforcement, error)
}
