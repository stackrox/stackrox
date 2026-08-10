package index

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	grpcPkg "github.com/stackrox/rox/pkg/grpc"
)

// Service provides an API to upsert virtual machine index reports to Central.
type Service interface {
	grpcPkg.APIService
	sensor.VirtualMachineIndexReportServiceServer
	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

// PullActiveChecker reports whether a VM is actively scraped via pull mode.
// When set, push-mode reports for actively-scraped VMs are dropped to avoid
// duplicates during the push→pull transition.
type PullActiveChecker interface {
	IsActivelyScraped(key string) bool
}

// NewService returns the VirtualMachineIndexReportServiceServer API for Sensor to use.
func NewService(handler Handler, pullChecker PullActiveChecker) Service {
	return &serviceImpl{handler: handler, pullChecker: pullChecker}
}
