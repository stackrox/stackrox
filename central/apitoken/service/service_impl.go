package service

import (
	"context"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/apitoken/backend"
	rolePkg "github.com/stackrox/rox/central/role"
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
	if err = verifyNoPrivilegeEscalation(utils.RoleNames(id.Roles()), req.GetRoles()); err != nil {
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
// The only two acceptable cases are:
// 	* principal has "Admin" role
// 	* all requested roles are assigned to principal
func verifyNoPrivilegeEscalation(userRoles, requestedRoles []string) error {
	userRoleNames := make(map[string]struct{}, len(userRoles))
	for _, userRole := range userRoles {
		userRoleNames[userRole] = struct{}{}
	}
	if _, ok := userRoleNames[rolePkg.Admin]; ok {
		return nil
	}
	for _, requestedRole := range requestedRoles {
		if _, ok := userRoleNames[requestedRole]; !ok {
			return errors.Wrapf(errorhelpers.ErrNotAuthorized,
				"requested API token roles: %s, but principal only has next roles: %s",
				strings.Join(requestedRoles, ", "),
				strings.Join(userRoles, ", "),
			)
		}
	}
	return nil
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
