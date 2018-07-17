package pipeline

import "bitbucket.org/stack-rox/apollo/generated/api/v1"

// Pipeline represents the processing applied to a SensorEvent to produce a response.
type Pipeline interface {
	Run(event *v1.SensorEvent) (*v1.SensorEventResponse, error)
}
