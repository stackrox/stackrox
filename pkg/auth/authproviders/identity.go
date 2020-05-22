package authproviders

import (
	"context"
	"errors"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/auth/user"
	"github.com/stackrox/rox/pkg/protoconv"
)

// CreateRoleBasedIdentity builds v1.AuthStatus containing identity and its role information from auth response
func CreateRoleBasedIdentity(ctx context.Context, provider Provider, authResp *AuthResponse) (*v1.AuthStatus, error) {
	if authResp == nil || authResp.Claims == nil {
		return nil, errors.New("authentication response is empty")
	}

	if provider == nil {
		return nil, errors.New("unexpected auth provider")
	}

	if provider.RoleMapper() == nil {
		return nil, errors.New("invalid role mapper")
	}

	ud := &permissions.UserDescriptor{
		UserID:     authResp.Claims.UserID,
		Attributes: authResp.Claims.Attributes,
	}

	roles, err := provider.RoleMapper().FromUserDescriptor(ctx, ud)
	if err != nil {
		return nil, err
	}

	// config might contain semi-sensitive values, so strip it
	var authProvider *storage.AuthProvider
	if provider.StorageView() != nil {
		authProvider = provider.StorageView()
		authProvider.Config = nil
	}

	return &v1.AuthStatus{
		Id: &v1.AuthStatus_UserId{
			UserId: authResp.Claims.UserID,
		},
		AuthProvider:   authProvider,
		Expires:        protoconv.ConvertTimeToTimestampOrNil(authResp.Expiration),
		UserAttributes: user.ConvertAttributes(authResp.Claims.Attributes),
		UserInfo:       getUserInfo(authResp.Claims, roles),
	}, nil
}

func getUserInfo(externalUserClaim *tokens.ExternalUserClaim, roles []*storage.Role) *storage.UserInfo {
	userInfo := &storage.UserInfo{
		Username:     externalUserClaim.UserID,
		FriendlyName: externalUserClaim.FullName,
		Permissions:  permissions.NewUnionRole(roles),
		Roles:        roles,
	}
	return userInfo
}
