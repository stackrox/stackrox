package service

import (
	"context"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/apitoken/backend"
	roleDS "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sliceutils"
	"google.golang.org/grpc"
)

const unrestricted = "Unrestricted"

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.APIToken)): {
			"/v1.APITokenService/GetAPIToken",
			"/v1.APITokenService/GetAPITokens",
		},
		user.With(permissions.Modify(resources.APIToken)): {
			"/v1.APITokenService/GenerateToken",
			"/v1.APITokenService/RevokeToken",
		},
	})
)

type serviceImpl struct {
	backend backend.Backend
	roles   roleDS.DataStore
}

func (s *serviceImpl) GetAPIToken(ctx context.Context, req *v1.ResourceByID) (*storage.TokenMetadata, error) {
	if req.GetId() == "" {
		return nil, errors.Wrap(errorhelpers.ErrInvalidArgs, "empty id passed")
	}
	token, err := s.backend.GetTokenOrNil(ctx, req.GetId())
	if err != nil {
		return nil, errors.Errorf("token retrieval failed: %s", err)
	}
	if token == nil {
		return nil, errors.Wrapf(errorhelpers.ErrInvalidArgs, "token with id '%s' does not exist", req.GetId())
	}
	return token, nil
}

func (s *serviceImpl) GetAPITokens(ctx context.Context, req *v1.GetAPITokensRequest) (*v1.GetAPITokensResponse, error) {
	tokens, err := s.backend.GetTokens(ctx, req)
	if err != nil {
		return nil, errors.Errorf("retrieval of tokens failed: %s", err)
	}
	return &v1.GetAPITokensResponse{
		Tokens: tokens,
	}, nil
}

func (s *serviceImpl) RevokeToken(ctx context.Context, req *v1.ResourceByID) (*v1.Empty, error) {
	exists, err := s.backend.RevokeToken(ctx, req.GetId())
	if err != nil {
		return &v1.Empty{}, errors.Errorf("couldn't revoke token: %s", err)
	}
	if !exists {
		return &v1.Empty{}, errors.Errorf("token with id '%s' does not exist", req.GetId())
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) GenerateToken(ctx context.Context, req *v1.GenerateTokenRequest) (*v1.GenerateTokenResponse, error) {
	if req.GetName() == "" {
		return nil, errors.Wrap(errorhelpers.ErrInvalidArgs, "token name cannot be empty")
	}

	if req.GetRole() != "" {
		if len(req.GetRoles()) > 0 {
			return nil, errors.Wrap(errorhelpers.ErrInvalidArgs, "must use either role or roles, but not both")
		}
		req.Roles = []string{req.GetRole()}
		req.Role = ""
	}

	roles, missingIndices, err := permissions.GetResolvedRolesFromStore(ctx, s.roles, req.GetRoles())
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch roles")
	}
	if len(missingIndices) > 0 {
		return nil, errors.Wrapf(errorhelpers.ErrInvalidArgs, "role(s) %s don't exist", strings.Join(sliceutils.StringSelect(req.GetRoles(), missingIndices...), ","))
	}

	id := authn.IdentityFromContext(ctx)
	if err = verifyNoPrivilegeEscalation(id.Roles(), roles); err != nil {
		return nil, err
	}

	token, metadata, err := s.backend.IssueRoleToken(ctx, req.GetName(), utils.RoleNames(roles))
	if err != nil {
		return nil, err
	}

	return &v1.GenerateTokenResponse{
		Token:    token,
		Metadata: metadata,
	}, nil
}

// This function ensures that no APIToken with permissions more than principal has can be created.
// For each requested tuple (access scope, resource, accessLevel) we check that either:
// * principal has permission on this resource with unrestricted access scope
// * principal has permission on this resource with requested access scope
func verifyNoPrivilegeEscalation(userRoles, requestedRoles []permissions.ResolvedRole) error {
	// Group roles by access scope.
	userRolesByScope := make(map[string][]permissions.ResolvedRole)
	accessScopeByName := make(map[string]*storage.SimpleAccessScope)
	userPermissionsByScope := make(map[string]map[string]storage.Access)
	for _, userRole := range userRoles {
		scopeName := extractScopeName(userRole)
		accessScopeByName[scopeName] = userRole.GetAccessScope()
		userRolesByScope[scopeName] = append(userRolesByScope[scopeName], userRole)
	}
	// Unify permissions of grouped roles.
	for scopeName, roles := range userRolesByScope {
		userPermissionsByScope[scopeName] = utils.NewUnionPermissions(roles)
	}

	// Verify that for each tuple (access scope, resource, accessLevel) we have enough permissions.
	var multiErr error
	for _, requestedRole := range requestedRoles {
		scopeName := extractScopeName(requestedRole)
		scopePermissions := userPermissionsByScope[scopeName]
		unrestrictedPermissions := userPermissionsByScope[unrestricted]
		for requestedResource, requestedAccess := range requestedRole.GetPermissions() {
			scopeAccess := scopePermissions[requestedResource]
			unrestrictedAccess := unrestrictedPermissions[requestedResource]
			maxAccess := utils.MaxAccess(scopeAccess, unrestrictedAccess)
			if maxAccess < requestedAccess {
				err := errors.Errorf("resource=%s, access scope=%s: requested access is %q, when maximum access is %q",
					requestedResource, scopeName, requestedAccess, maxAccess)
				multiErr = multierror.Append(multiErr, err)
			}
		}
	}
	return multiErr
}

func extractScopeName(userRole permissions.ResolvedRole) string {
	if userRole.GetAccessScope() != nil {
		return userRole.GetAccessScope().GetName()
	}
	return unrestricted
}

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterAPITokenServiceServer(grpcServer, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterAPITokenServiceHandler(ctx, mux, conn)
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}
