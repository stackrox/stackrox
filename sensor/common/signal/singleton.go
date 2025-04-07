package signal

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sensor/queue"
)

func NewService(opts ...Option) Service {
	queueSize := queue.ScaleSizeOnNonDefault(env.ProcessSignalQueueSize)
	srv := &serviceImpl{
		queue:            make(chan *storage.ProcessSignal, queueSize),
		authFuncOverride: authFuncOverride,
	}

	for _, o := range opts {
		o(srv)
	}
	return srv
}
