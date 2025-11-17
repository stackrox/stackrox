package service

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
)

// Service that the fact agent can send information to
type Service interface {
	pkgGRPC.APIService
	sensor.FileActivityServiceServer

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}
