package service

import (
	"context"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service is the service
type Service interface {
	grpc.APIService
	v2.ComplianceProfileServiceServer

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}
