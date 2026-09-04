package signal

import (
	"github.com/stackrox/rox/sensor/common/message"
)

// New creates a new signal service
func New(pipeline Pipeline, indicators chan *message.ExpiringMessage, opts ...Option) Service {
	srv := &serviceImpl{
		indicators:       indicators,
		processPipeline:  pipeline,
		authFuncOverride: authFuncOverride,
		writer:           nil,
	}
	for _, o := range opts {
		o(srv)
	}
	return srv
}
