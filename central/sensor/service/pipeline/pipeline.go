package pipeline

import (
	"github.com/stackrox/rox/generated/internalapi/central"
)

// Pipeline represents the processing applied to a SensorEvent to produce a response.
//go:generate mockgen-wrapper Pipeline
type Pipeline interface {
	Run(msg *central.MsgFromSensor, injector MsgInjector) error
}

// Factory returns a Pipeline for the given cluster.
type Factory interface {
	GetPipeline(clusterID string) (Pipeline, error)
}

// Fragment is a component of a Pipeline that only processes specific messages.
//go:generate mockgen-wrapper Fragment
type Fragment interface {
	Pipeline
	Match(msg *central.MsgFromSensor) bool
}

// FragmentFactory returns a Fragment for the given cluster.
type FragmentFactory interface {
	GetFragment(clusterID string) (Fragment, error)
}

// MsgInjector allows a pipeline to return a MsgToSensor back into the pipeline.
// It does a best-effort send, and returns a bool whether it succeeded. (It will fail if the stream in the
// time between the object being passed to the pipeline and the stream being broken.)
type MsgInjector interface {
	InjectMessage(msg *central.MsgToSensor) bool
}
