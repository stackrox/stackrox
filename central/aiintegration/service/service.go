package service

import (
	"context"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service is the interface for the AI integration gRPC service.
type Service interface {
	grpc.APIService
	v2.AiIntegrationServiceServer

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}
