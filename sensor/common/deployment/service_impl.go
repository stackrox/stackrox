package deployment

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	grpcPkg "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/sensor/common/store"
	"google.golang.org/grpc"
)

// Service is an interface provides functionality to get deployments from Sensor.
type Service interface {
	grpcPkg.APIService
	sensor.DeploymentServiceServer
	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

// NewService returns the DeploymentServiceServer API for Sensor to use.
func NewService(deployments store.DeploymentStore, pods store.PodStore) Service {
	return &serviceImpl{
		deployments: deployments,
		pods:        pods,
	}
}

type serviceImpl struct {
	sensor.UnimplementedDeploymentServiceServer

	deployments store.DeploymentStore
	pods        store.PodStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	sensor.RegisterDeploymentServiceServer(grpcServer, s)
}

// RegisterServiceHandler implements the APIService interface, but the agent does not accept calls over the gRPC gateway
func (s *serviceImpl) RegisterServiceHandler(context.Context, *runtime.ServeMux, *grpc.ClientConn) error {
	return nil
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, idcheck.AdmissionControlOnly().Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetDeploymentForPod(_ context.Context, req *sensor.GetDeploymentForPodRequest) (*storage.Deployment, error) {
	if req.GetPodName() == "" || req.GetNamespace() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "pod namespace and pod name must be provided")
	}

	pod := s.pods.GetByName(req.GetPodName(), req.GetNamespace())
	if pod == nil {
		return nil, errors.Wrapf(errox.NotFound,
			"namespace/%s/pods/%s not found",
			req.GetNamespace(), req.GetPodName())
	}

	dep := s.deployments.Get(pod.GetDeploymentId())
	if dep == nil {
		return nil, errors.Wrapf(errox.NotFound,
			"no containing deployment found for namespace/%s/pods/%s",
			req.GetNamespace(), req.GetPodName())
	}
	return dep, nil
}
