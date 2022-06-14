package service

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service is the service
type Service interface {
	grpc.APIService
	v1.ComplianceServiceServer

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}
