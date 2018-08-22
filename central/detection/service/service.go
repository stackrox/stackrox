package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	buildTimeDetection "github.com/stackrox/rox/central/detection/buildtime"
	runTimeDetection "github.com/stackrox/rox/central/detection/runtime"
	"github.com/stackrox/rox/generated/api/v1"
	"google.golang.org/grpc"
)

// Service provides the interface for running detection on images and containers.
type Service interface {
	RegisterServiceServer(grpcServer *grpc.Server)
	RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	DetectBuildTime(ctx context.Context, request *v1.Image) (*v1.DetectionResponse, error)
	DetectRunTime(ctx context.Context, request *v1.Container) (*v1.DetectionResponse, error)
}

// New returns a new Service instance using the given DataStore.
func New(buildTimeDetector buildTimeDetection.Detector, runTimeDetector runTimeDetection.Detector) Service {
	return &serviceImpl{
		buildTimeDetector: buildTimeDetector,
		runTimeDetector:   runTimeDetector,
	}
}
