package signal

import (
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/signal/component"
)

func NewComponent(pipeline component.Pipeline, indicators chan *message.ExpiringMessage, opts ...component.Option) common.SensorComponent {
	return component.New(pipeline, indicators, opts...)
}

func NewService(queue chan *sensor.ProcessSignal, opts ...Option) Service {
	srv := &serviceImpl{
		queue:            queue,
		authFuncOverride: authFuncOverride,
	}

	for _, o := range opts {
		o(srv)
	}
	return srv
}
