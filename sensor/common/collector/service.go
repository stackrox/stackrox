package collector

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/grpc"
)

type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

func NewService(queue chan *sensor.ProcessSignal, opts ...Option) Service {
	return newService(queue)
}
