package service

import (
	"context"

	"github.com/stackrox/stackrox/generated/internalapi/sensor"
	pkgGRPC "github.com/stackrox/stackrox/pkg/grpc"
	"github.com/stackrox/stackrox/pkg/logging"
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
