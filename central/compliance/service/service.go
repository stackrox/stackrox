package service

import (
	"context"

	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/grpc"
)

// Service is the service
type Service interface {
	grpc.APIService
	v1.ComplianceServiceServer

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}
