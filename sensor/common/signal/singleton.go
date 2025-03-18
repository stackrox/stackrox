package signal

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/signal/component"
)

func NewComponent(pipeline component.Pipeline, indicators chan *message.ExpiringMessage, opts ...component.Option) common.SensorComponent {
	return component.New(pipeline, indicators, opts...)
}

func NewService(queue chan *v1.Signal, opts ...Option) Service {
	srv := &serviceImpl{
		queue:            queue,
		authFuncOverride: authFuncOverride,
	}

	for _, o := range opts {
		o(srv)
	}
	return srv
}
