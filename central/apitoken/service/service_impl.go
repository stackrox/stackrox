package service

import (
	"context"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/apitoken/backend"
	roleDS "github.com/stackrox/rox/central/role/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
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

	id, err := authn.IdentityFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := verifyNoPrivilegeEscalation(id.Roles(), roles); err != nil {
		return nil, errox.NewErrNotAuthorized(err.Error())
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

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterAPITokenServiceServer(grpcServer, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterAPITokenServiceHandler(ctx, mux, conn)
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}
