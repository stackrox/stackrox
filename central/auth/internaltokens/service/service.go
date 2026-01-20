package service

import (
	"context"

	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to serve tokens for internal purposes.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}
