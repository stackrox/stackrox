package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/clusterinit/backend"
	"github.com/stackrox/rox/central/role"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"google.golang.org/grpc"
)

var (
	authorizer = user.WithRole(role.Admin)
)

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
		return nil, errors.Wrap(err, "retrieving meta data for all init bundles")
	}

	v1InitBundleMetas := make([]*v1.InitBundleMeta, 0, len(initBundleMetas))
	for _, initBundle := range initBundleMetas {
		v1InitBundleMetas = append(v1InitBundleMetas, InitBundleMetaStorageToV1(initBundle))
	}

	return &v1.InitBundleMetasResponse{Items: v1InitBundleMetas}, nil
}

func (s *serviceImpl) GenerateInitBundle(ctx context.Context, request *v1.InitBundleGenRequest) (*v1.InitBundleGenResponse, error) {
	generated, err := s.backend.Issue(ctx, request.GetName())
	if err != nil {
		return nil, errors.Wrap(err, "generating new init bundle")
	}
	meta := InitBundleMetaStorageToV1(generated.Meta)

	bundleYaml, err := generated.RenderAsYAML()
	if err != nil {
		return nil, errors.Wrap(err, "rendering init bundle as YAML")
	}
	return &v1.InitBundleGenResponse{
		HelmValuesBundle: bundleYaml,
		Meta:             meta,
	}, nil
}
