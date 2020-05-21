package signal

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
)

// New creates a new signal service
func New(pipeline Pipeline, indicators chan *central.MsgFromSensor) Service {
	return &serviceImpl{
		queue:           make(chan *v1.Signal, maxBufferSize),
		indicators:      indicators,
		processPipeline: pipeline,
	}
}
