package authproviders

import (
	"context"
	"errors"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/auth/user"
	"github.com/stackrox/rox/pkg/protoconv"
)

// CreateRoleBasedIdentity builds v1.AuthStatus containing identity and its role information from auth response.
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

	// config might contain semi-sensitive values, so strip it
	var authProvider *storage.AuthProvider
	if provider.StorageView() != nil {
		authProvider = provider.StorageView().Clone()
		authProvider.Config = nil
	}

	resolvedRoles, err := provider.RoleMapper().FromUserDescriptor(ctx, ud)
	if err != nil {
		return nil, err
	}
	return &v1.AuthStatus{
		Id: &v1.AuthStatus_UserId{
			UserId: authResp.Claims.UserID,
		},
		AuthProvider:   authProvider,
		Expires:        protoconv.ConvertTimeToTimestampOrNil(authResp.Expiration),
		UserAttributes: user.ConvertAttributes(authResp.Claims.Attributes),
		UserInfo:       getUserInfo(authResp.Claims, resolvedRoles),
	}, nil
}

func getUserInfo(externalUserClaim *tokens.ExternalUserClaim, resolvedRoles []permissions.ResolvedRole) *storage.UserInfo {
	userInfo := &storage.UserInfo{
		Username:     externalUserClaim.UserID,
		FriendlyName: externalUserClaim.FullName,
		Permissions:  &storage.UserInfo_ResourceToAccess{ResourceToAccess: utils.NewUnionPermissions(resolvedRoles)},
		Roles:        utils.ExtractRolesForUserInfo(resolvedRoles),
	}
	return userInfo
}
