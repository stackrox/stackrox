package service

import (
	"context"

	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the microservice that serves tokens for the OCP plugin.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}
