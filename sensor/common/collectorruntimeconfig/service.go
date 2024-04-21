package collectorruntimeconfig

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/sensor/common"
)

//go:generate mockgen-wrapper Service

// Service is an interface to receiving CollectorReturns from launched daemons.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	sensor.CollectorServiceServer
	common.SensorComponent

	// RunScrape(msg *sensor.MsgToCollector) int
}

// NewService returns the CollectorServiceServer API for Sensor to use.
func NewService(collectorC chan common.MessageToCollectorWithAddress) Service {
	return &serviceImpl{
		collectorC:        collectorC,
		connectionManager: newConnectionManager(),
	}
}
