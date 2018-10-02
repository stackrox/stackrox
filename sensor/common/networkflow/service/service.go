package service

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Service that the Collector can send network connection info to.
type Service interface {
	pkgGRPC.APIService
	sensor.NetworkConnectionInfoServiceServer

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}
