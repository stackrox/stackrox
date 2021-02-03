package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/clusterinit/backend"
	"github.com/stackrox/rox/central/clusterinit/store"
	"github.com/stackrox/rox/central/role"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = user.WithRole(role.Admin)
)

var _ v1.ClusterInitServiceServer = (*serviceImpl)(nil)

type serviceImpl struct {
	backend backend.Backend
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterClusterInitServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterClusterInitServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetInitBundles(ctx context.Context, empty *v1.Empty) (*v1.InitBundleMetasResponse, error) {
	initBundleMetas, err := s.backend.GetAll(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "retrieving meta data for all init bundles: %s", err)
	}

	v1InitBundleMetas := make([]*v1.InitBundleMeta, 0, len(initBundleMetas))
	for _, initBundle := range initBundleMetas {
		v1InitBundleMetas = append(v1InitBundleMetas, InitBundleMetaStorageToV1(initBundle))
	}

	return &v1.InitBundleMetasResponse{Items: v1InitBundleMetas}, nil
}

func (s *serviceImpl) GetCAConfig(ctx context.Context, _ *v1.Empty) (*v1.GetCAConfigResponse, error) {
	caConfig, err := s.backend.GetCAConfig(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "retrieving meta data for all ")
	}

	caConfigYAML, err := caConfig.RenderAsYAML()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to render CA config to YAML: %v", err)
	}

	return &v1.GetCAConfigResponse{
		HelmValuesBundle: caConfigYAML,
	}, nil
}

func (s *serviceImpl) GenerateInitBundle(ctx context.Context, request *v1.InitBundleGenRequest) (*v1.InitBundleGenResponse, error) {
	generated, err := s.backend.Issue(ctx, request.GetName())
	if err != nil {
		if errors.Is(err, store.ErrInitBundleDuplicateName) {
			return nil, status.Errorf(codes.AlreadyExists, "generating new init bundle: %s", err)
		}
		return nil, status.Errorf(codes.Internal, "generating new init bundle: %s", err)
	}
	meta := InitBundleMetaStorageToV1(generated.Meta)

	bundleYaml, err := generated.RenderAsYAML()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "rendering init bundle as YAML: %s", err)
	}
	bundleK8sManifest, err := generated.RenderAsK8sSecrets()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "rendering init bundle as Kubernetes secrets: %s", err)
	}

	return &v1.InitBundleGenResponse{
		HelmValuesBundle: bundleYaml,
		KubectlBundle:    bundleK8sManifest,
		Meta:             meta,
	}, nil
}

func (s *serviceImpl) RevokeInitBundle(ctx context.Context, request *v1.InitBundleRevokeRequest) (*v1.Empty, error) {
	for _, id := range request.GetIds() {
		if err := s.backend.Revoke(ctx, id); err != nil {
			if errors.Is(err, store.ErrInitBundleNotFound) {
				return nil, status.Errorf(codes.NotFound, "revoking %q failed: %s", id, err)
			}
			return nil, status.Errorf(codes.Internal, "revoking %q failed: %s", id, err)
		}
	}
	return &v1.Empty{}, nil
}
