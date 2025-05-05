package signal

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
)

func NewService(opts ...Option) Service {
	srv := &serviceImpl{
		queue:            make(chan *storage.ProcessSignal, env.ProcessSignalChannelBufferSize.IntegerSetting()),
		authFuncOverride: authFuncOverride,
	}

	for _, o := range opts {
		o(srv)
	}
	return srv
}
