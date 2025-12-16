package service

import (
	"context"

	"github.com/stackrox/rox/central/apitoken/backend"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the service that issues scoped tokens.
type Service interface {
	v1.ScopedTokenServiceServer

	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

// New returns a ready-to-use instance of Service.
func New(tokenBackend backend.Backend) Service {
	return &serviceImpl{tokenBackend: tokenBackend}
}
