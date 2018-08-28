package service

import (
	"context"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	buildtimeDetection "github.com/stackrox/rox/central/detection/buildtime"
	runtimeDetection "github.com/stackrox/rox/central/detection/runtime"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/service"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/images/utils"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Detection)): {
			"/v1.DetectionService/DetectBuildTime",
			"/v1.DetectionService/DetectRunTime",
		},
	})
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	imageEnricher     enricher.ImageEnricher
	buildTimeDetector buildtimeDetection.Detector
	runTimeDetector   runtimeDetection.Detector
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterDetectionServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterDetectionServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, service.ReturnErrorCode(authorizer.Authorized(ctx, fullMethodName))
}

// DetectBuildTime runs detection on a built image.
func (s *serviceImpl) DetectBuildTime(ctx context.Context, image *v1.Image) (*v1.DetectionResponse, error) {
	if image.Name == nil {
		return nil, fmt.Errorf("image name contents missing")
	}
	utils.FillFullName(image.Name)

	_ = s.imageEnricher.EnrichImage(image)

	alerts, err := s.buildTimeDetector.Detect(image)
	if err != nil {
		return nil, err
	}
	return &v1.DetectionResponse{
		Alerts: alerts,
	}, nil
}

// DetectRunTime runs detection on a running container.
func (s *serviceImpl) DetectRunTime(ctx context.Context, container *v1.Container) (*v1.DetectionResponse, error) {
	alerts, err := s.runTimeDetector.Detect(container)
	if err != nil {
		return nil, err
	}
	return &v1.DetectionResponse{
		Alerts: alerts,
	}, nil
}
