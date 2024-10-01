package service

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/concurrency"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
)

//go:generate mockgen-wrapper

// Service that the Collector can send network connection info to.
type Service interface {
	pkgGRPC.APIService
	sensor.NetworkConnectionInfoServiceServer

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
	SendCollectorConfig(stream sensor.NetworkConnectionInfoService_PushNetworkConnectionInfoServer, iter concurrency.ValueStreamIter[*sensor.CollectorConfig]) error
}
