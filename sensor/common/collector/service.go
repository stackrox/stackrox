package collector

import (
	"context"

	"github.com/stackrox/rox/pkg/grpc"
)

type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

func NewService(opts ...Option) Service {
	return newService()
}
