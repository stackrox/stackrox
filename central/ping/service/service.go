package service

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
	Ping(context.Context, *v1.Empty) (*v1.PongMessage, error)
}

// New returns a new Service instance using the given DataStore.
func New() Service {
	return &serviceImpl{}
}
