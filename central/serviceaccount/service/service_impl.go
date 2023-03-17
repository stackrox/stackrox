package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	deploymentStore "github.com/stackrox/rox/central/deployment/datastore"
	namespaceStore "github.com/stackrox/rox/central/namespace/datastore"
	roleDatastore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	bindingDatastore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	"github.com/stackrox/rox/central/rbac/utils"
	saDatastore "github.com/stackrox/rox/central/serviceaccount/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/k8srbac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.ServiceAccount)): {
			"/v1.ServiceAccountService/GetServiceAccount",
			"/v1.ServiceAccountService/ListServiceAccounts",
		},
	})
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	v1.UnimplementedServiceAccountServiceServer

	serviceAccounts saDatastore.DataStore
	bindings        bindingDatastore.DataStore
	roles           roleDatastore.DataStore
	deployments     deploymentStore.DataStore
	namespaces      namespaceStore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterServiceAccountServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterServiceAccountServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetServiceAccount returns the service account for the id.
func (s *serviceImpl) GetServiceAccount(ctx context.Context, request *v1.ResourceByID) (*v1.GetServiceAccountResponse, error) {
	sa, exists, err := s.serviceAccounts.GetServiceAccount(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "service account with id '%s' does not exist", request.GetId())
	}

	clusterRoles, scopedRoles, err := s.getRoles(ctx, sa)

	if err != nil {
		return nil, err
	}

	return &v1.GetServiceAccountResponse{
		SaAndRole: &v1.ServiceAccountAndRoles{
			ServiceAccount:          sa,
			ClusterRoles:            clusterRoles,
			ScopedRoles:             scopedRoles,
			DeploymentRelationships: s.getDeploymentRelationships(ctx, sa),
		},
	}, nil
}

// ListServiceAccounts returns all service accounts that match the query.
func (s *serviceImpl) ListServiceAccounts(ctx context.Context, rawQuery *v1.RawQuery) (*v1.ListServiceAccountResponse, error) {
	q, err := search.ParseQuery(rawQuery.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	serviceAccounts, err := s.serviceAccounts.SearchRawServiceAccounts(ctx, q)

	if err != nil {
		return nil, errors.Errorf("failed to retrieve service accounts: %s", err)
	}

	saAndRoles := make([]*v1.ServiceAccountAndRoles, 0, len(serviceAccounts))
	for _, sa := range serviceAccounts {
		clusterRoles, scopedRoles, err := s.getRoles(ctx, sa)

		if err != nil {
			return nil, err
		}

		saAndRoles = append(saAndRoles, &v1.ServiceAccountAndRoles{
			ServiceAccount:          sa,
			ClusterRoles:            clusterRoles,
			ScopedRoles:             scopedRoles,
			DeploymentRelationships: s.getDeploymentRelationships(ctx, sa),
		})

	}
	return &v1.ListServiceAccountResponse{
		SaAndRoles: saAndRoles,
	}, nil
}

func (s *serviceImpl) getDeploymentRelationships(ctx context.Context, sa *storage.ServiceAccount) []*v1.SADeploymentRelationship {
	psr := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, sa.GetClusterId()).
		AddExactMatches(search.Namespace, sa.GetNamespace()).
		AddExactMatches(search.ServiceAccountName, sa.GetName()).
		ProtoQuery()

	deploymentResults, err := s.deployments.SearchListDeployments(ctx, psr)
	if err != nil {
		return nil
	}

	deployments := make([]*v1.SADeploymentRelationship, 0, len(deploymentResults))
	for _, r := range deploymentResults {
		deployments = append(deployments, &v1.SADeploymentRelationship{
			Id:   r.Id,
			Name: r.Name,
		})
	}

	return deployments
}

func (s *serviceImpl) getRoles(ctx context.Context, sa *storage.ServiceAccount) ([]*storage.K8SRole, []*v1.ScopedRoles, error) {
	subject := k8srbac.GetSubjectForServiceAccount(sa)

	clusterEvaluator := utils.NewClusterPermissionEvaluator(sa.GetClusterId(), s.roles, s.bindings)
	clusterRoles := clusterEvaluator.RolesForSubject(ctx, subject)

	namespaceQuery := search.NewQueryBuilder().AddExactMatches(search.ClusterID, sa.ClusterId).ProtoQuery()
	namespaces, err := s.namespaces.SearchNamespaces(ctx, namespaceQuery)
	if err != nil {
		return clusterRoles, nil, err
	}

	scopedRoles := make([]*v1.ScopedRoles, 0)
	for _, namespace := range namespaces {
		namespaceEvaluator := utils.NewNamespacePermissionEvaluator(sa.ClusterId, namespace.GetName(), s.roles, s.bindings)
		namespaceRoles := namespaceEvaluator.RolesForSubject(ctx, subject)

		if len(namespaceRoles) != 0 {
			scopedRoles = append(scopedRoles, &v1.ScopedRoles{
				Namespace: namespace.GetName(),
				Roles:     namespaceRoles,
			})
		}
	}

	return clusterRoles, scopedRoles, nil
}
