package service

import (
	"context"
	"math"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/namespace"
	"github.com/stackrox/rox/central/namespace/datastore"
	npDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"google.golang.org/grpc"
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
	v1.UnimplementedNamespaceServiceServer

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

func (s *serviceImpl) GetNamespaces(ctx context.Context, req *v1.GetNamespaceRequest) (*v1.GetNamespacesResponse, error) {
	rawQuery := req.GetQuery()
	parsedQuery, err := search.ParseQuery(rawQuery.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	// Fill in pagination. MaxInt32 preserves previous functionality
	paginated.FillPagination(parsedQuery, rawQuery.GetPagination(), math.MaxInt32)

	namespaces, err := namespace.ResolveAll(ctx, s.datastore, s.deployments, s.secrets, s.networkPolicies, parsedQuery)
	if err != nil {
		return nil, errors.Errorf("Failed to retrieve namespaces: %v", err)
	}
	return &v1.GetNamespacesResponse{
		Namespaces: namespaces,
	}, nil
}

func (s *serviceImpl) GetNamespace(ctx context.Context, req *v1.ResourceByID) (*v1.Namespace, error) {
	if req.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "ID cannot be empty")
	}
	resolvedNS, found, err := namespace.ResolveByID(ctx, req.GetId(), s.datastore, s.deployments, s.secrets, s.networkPolicies)
	if err != nil {
		return nil, errors.Errorf("Failed to retrieve namespace: %v", err)
	}
	if !found {
		return nil, errors.Wrapf(errox.InvalidArgs, "Namespace '%s' not found", req.GetId())
	}
	return resolvedNS, nil
}

func (*serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}
