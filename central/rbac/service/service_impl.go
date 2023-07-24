package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	rolesDataStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	roleBindingsDataStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.K8sRole)): {
			"/v1.RbacService/GetRole",
			"/v1.RbacService/ListRoles",
		},
		user.With(permissions.View(resources.K8sRoleBinding)): {
			"/v1.RbacService/GetRoleBinding",
			"/v1.RbacService/ListRoleBindings",
		},
		user.With(permissions.View(resources.K8sSubject)): {
			"/v1.RbacService/GetSubject",
			"/v1.RbacService/ListSubjects",
		},
	})
)

// serviceImpl provides APIs for k8s rbac objects.
type serviceImpl struct {
	v1.UnimplementedRbacServiceServer

	roles    rolesDataStore.DataStore
	bindings roleBindingsDataStore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterRbacServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterRbacServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetRole returns the k8s role for the id.
func (s *serviceImpl) GetRole(ctx context.Context, request *v1.ResourceByID) (*v1.GetRoleResponse, error) {
	role, exists, err := s.roles.GetRole(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "k8s role with id '%q' does not exist", request.GetId())
	}

	return &v1.GetRoleResponse{Role: role}, nil
}

// ListRoles returns all roles that match the query.
func (s *serviceImpl) ListRoles(ctx context.Context, rawQuery *v1.RawQuery) (*v1.ListRolesResponse, error) {
	// TODO: Link policy rule fields? I.E. if query has Verbs:Get,Resource:Pods, we want the two linked so only
	// roles that can get pods are returned, not roles that can get anything, and can do any operation on Pods.
	q, err := search.ParseQuery(rawQuery.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	roles, err := s.roles.SearchRawRoles(ctx, q)
	if err != nil {
		return nil, errors.Errorf("failed to retrieve k8s roles: %v", err)
	}

	return &v1.ListRolesResponse{Roles: roles}, nil
}

// GetRole returns the k8s role binding for the id.
func (s *serviceImpl) GetRoleBinding(ctx context.Context, request *v1.ResourceByID) (*v1.GetRoleBindingResponse, error) {
	binding, exists, err := s.bindings.GetRoleBinding(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "k8s role binding with id '%q' does not exist", request.GetId())
	}

	return &v1.GetRoleBindingResponse{Binding: binding}, nil
}

// ListRoleBindings returns all role bindings that match the query.
func (s *serviceImpl) ListRoleBindings(ctx context.Context, rawQuery *v1.RawQuery) (*v1.ListRoleBindingsResponse, error) {
	q, err := search.ParseQuery(rawQuery.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	bindings, err := s.bindings.SearchRawRoleBindings(ctx, q)
	if err != nil {
		return nil, err
	}
	return &v1.ListRoleBindingsResponse{Bindings: bindings}, nil
}

// GetSubject returns the subject with the input ID (the unique subject name).
func (s *serviceImpl) GetSubject(ctx context.Context, request *v1.ResourceByID) (*v1.GetSubjectResponse, error) {
	return getSubjectFromStores(ctx, request.GetId(), s.roles, s.bindings)
}

// ListSubjects returns all of the subjects granted roles that match the input query.
func (s *serviceImpl) ListSubjects(ctx context.Context, rawQuery *v1.RawQuery) (*v1.ListSubjectsResponse, error) {
	// Keep only binding specific fields in the query.
	bindingQuery := &v1.RawQuery{
		Query: search.FilterFields(rawQuery.GetQuery(), func(field string) bool {
			_, isBindingField := schema.RoleBindingsSchema.OptionsMap.Get(field)
			return isBindingField
		}),
	}
	bindingsSearch, err := s.ListRoleBindings(ctx, bindingQuery)
	if err != nil {
		return nil, err
	}

	// Use all roles (filtered bindings will do what we need).
	roleSearch, err := s.ListRoles(ctx, &v1.RawQuery{})
	if err != nil {
		return nil, err
	}

	// List all of the subjects.
	return listSubjects(rawQuery, roleSearch.GetRoles(), bindingsSearch.GetBindings())
}
