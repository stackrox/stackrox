package signal

import (
	"github.com/stackrox/rox/generated/storage"
)

const maxBufferSize = 10000

func NewService(opts ...Option) Service {
	srv := &serviceImpl{
		queue:            make(chan *storage.ProcessSignal, maxBufferSize),
		authFuncOverride: authFuncOverride,
	}

	for _, o := range opts {
		o(srv)
	}
	return srv
}
