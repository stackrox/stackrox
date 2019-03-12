package service

import (
	"context"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/detection/buildtime"
	"github.com/stackrox/rox/central/detection/deploytime"
	"github.com/stackrox/rox/central/enrichment"
	"github.com/stackrox/rox/central/role/resources"
	apiV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/logging"
	resourcesConv "github.com/stackrox/rox/pkg/protoconv/resources"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	coreV1 "k8s.io/api/core/v1"
	k8sRuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Detection)): {
			"/v1.DetectionService/DetectBuildTime",
			"/v1.DetectionService/DetectDeployTimeFromYAML",
		},
		or.SensorOrAuthorizer(user.With(permissions.Modify(resources.Detection))): {
			"/v1.DetectionService/DetectDeployTime",
		},
	})

	log = logging.LoggerForModule()
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	imageEnricher      enricher.ImageEnricher
	deploymentEnricher enrichment.Enricher
	buildTimeDetector  buildtime.Detector

	detector deploytime.Detector
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	apiV1.RegisterDetectionServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return apiV1.RegisterDetectionServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// DetectBuildTime runs detection on a built image.
func (s *serviceImpl) DetectBuildTime(ctx context.Context, req *apiV1.BuildDetectionRequest) (*apiV1.BuildDetectionResponse, error) {
	if req.GetImage().GetName() == nil {
		return nil, fmt.Errorf("image name contents missing")
	}

	_ = s.imageEnricher.EnrichImage(enricher.EnrichmentContext{FastPath: req.GetFastPath()}, req.GetImage())

	alerts, err := s.buildTimeDetector.Detect(req.GetImage())
	if err != nil {
		return nil, err
	}
	return &apiV1.BuildDetectionResponse{
		Alerts: alerts,
	}, nil
}

func (s *serviceImpl) enrichAndDetect(ctx enricher.EnrichmentContext, deployment *storage.Deployment) (*apiV1.DeployDetectionResponse_Run, error) {
	_, _, err := s.deploymentEnricher.EnrichDeployment(ctx, deployment)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	alerts, err := s.detector.Detect(deployment)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &apiV1.DeployDetectionResponse_Run{
		Name:   deployment.GetName(),
		Type:   deployment.GetType(),
		Alerts: alerts,
	}, nil
}

func (s *serviceImpl) runDeployTimeDetect(ctx enricher.EnrichmentContext, obj k8sRuntime.Object) (*apiV1.DeployDetectionResponse_Run, error) {
	if !kubernetes.IsDeploymentResource(obj.GetObjectKind().GroupVersionKind().Kind) {
		return nil, nil
	}

	deployment, err := resourcesConv.NewDeploymentFromStaticResource(obj, obj.GetObjectKind().GroupVersionKind().Kind)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Could not convert to deployment from resource: %v", err)
	}

	return s.enrichAndDetect(ctx, deployment)
}

// DetectDeployTime runs detection on a deployment
func (s *serviceImpl) DetectDeployTimeFromYAML(ctx context.Context, req *apiV1.DeployYAMLDetectionRequest) (*apiV1.DeployDetectionResponse, error) {
	if req.GetYaml() == "" {
		return nil, status.Error(codes.InvalidArgument, "yaml field must be specified in detection request")
	}
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(req.GetYaml()), nil, nil)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "could not parse YAML: %v", err)
	}

	eCtx := enricher.EnrichmentContext{FastPath: req.GetFastPath()}
	var runs []*apiV1.DeployDetectionResponse_Run
	if list, ok := obj.(*coreV1.List); ok {
		for i, item := range list.Items {
			o2, _, err := decode(item.Raw, nil, nil)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "Could not decode item %d in the list: %v", i, err)
			}
			run, err := s.runDeployTimeDetect(eCtx, o2)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Unable to convert item %d in the list: %v", i, err)
			}
			if run != nil {
				runs = append(runs, run)
			}
		}
	} else {
		run, err := s.runDeployTimeDetect(eCtx, obj)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Unable to convert object: %v", err)
		}
		if run != nil {
			runs = append(runs, run)
		}
	}
	return &apiV1.DeployDetectionResponse{
		Runs: runs,
	}, nil
}

func (s *serviceImpl) DetectDeployTime(ctx context.Context, req *apiV1.DeployDetectionRequest) (*apiV1.DeployDetectionResponse, error) {
	if req.GetDeployment() == nil {
		return nil, status.Error(codes.InvalidArgument, "Deployment must be passed to deploy time detection")
	}

	run, err := s.enrichAndDetect(enricher.EnrichmentContext{FastPath: req.GetFastPath()}, req.GetDeployment())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &apiV1.DeployDetectionResponse{
		Runs: []*apiV1.DeployDetectionResponse_Run{
			run,
		},
	}, nil
}
