package service

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service gRPC connection preferences from central.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
	Get(ctx context.Context, empty *v1.Empty) (*v1.Preferences, error)
}

// New returns an instance of the gRPC preferences API service.
func New() Service {
	return &serviceImpl{}
}
