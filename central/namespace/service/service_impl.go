package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/namespace"
	"github.com/stackrox/rox/central/namespace/datastore"
	npDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	"github.com/stackrox/rox/central/role/resources"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Namespace)): {
			"/v1.NamespaceService/GetNamespace",
			"/v1.NamespaceService/GetNamespaces",
		},
	})
)

type serviceImpl struct {
	datastore       datastore.DataStore
	deployments     deploymentDataStore.DataStore
	secrets         secretDataStore.DataStore
	networkPolicies npDS.DataStore
}

func (s *serviceImpl) RegisterServiceServer(server *grpc.Server) {
	v1.RegisterNamespaceServiceServer(server, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterNamespaceServiceHandler(ctx, mux, conn)
}

func (s *serviceImpl) GetNamespaces(ctx context.Context, _ *v1.Empty) (*v1.GetNamespacesResponse, error) {
	namespaces, err := namespace.ResolveAll(ctx, s.datastore, s.deployments, s.secrets, s.networkPolicies)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to retrieve namespaces: %v", err)
	}
	return &v1.GetNamespacesResponse{
		Namespaces: namespaces,
	}, nil
}

func (s *serviceImpl) GetNamespace(ctx context.Context, req *v1.ResourceByID) (*v1.Namespace, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "ID cannot be empty")
	}
	resolvedNS, found, err := namespace.ResolveByID(ctx, req.GetId(), s.datastore, s.deployments, s.secrets, s.networkPolicies)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to retrieve namespace: %v", err)
	}
	if !found {
		return nil, status.Errorf(codes.InvalidArgument, "Namespace '%s' not found", req.GetId())
	}
	return resolvedNS, nil
}

func (*serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}
