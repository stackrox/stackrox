package collector

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/grpc"
)

type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
	GetMessagesC() <-chan *sensor.ProcessSignal
}

func NewService(opts ...Option) Service {
	return newService(opts...)
}
