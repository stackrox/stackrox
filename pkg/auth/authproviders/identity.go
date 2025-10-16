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
		IdpToken:   authResp.IdpToken,
	}

	// config might contain semi-sensitive values, so strip it
	var authProvider *storage.AuthProvider
	if provider.StorageView() != nil {
		authProvider = provider.StorageView().CloneVT()
		authProvider.SetConfig(nil)
	}

	resolvedRoles, err := provider.RoleMapper().FromUserDescriptor(ctx, ud)
	if err != nil {
		return nil, err
	}
	as := &v1.AuthStatus{}
	as.SetUserId(authResp.Claims.UserID)
	as.SetAuthProvider(authProvider)
	as.SetExpires(protoconv.ConvertTimeToTimestampOrNil(authResp.Expiration))
	as.SetUserAttributes(user.ConvertAttributes(authResp.Claims.Attributes))
	as.SetUserInfo(getUserInfo(authResp.Claims, resolvedRoles))
	return as, nil
}

func getUserInfo(externalUserClaim *tokens.ExternalUserClaim, resolvedRoles []permissions.ResolvedRole) *storage.UserInfo {
	ur := &storage.UserInfo_ResourceToAccess{}
	ur.SetResourceToAccess(utils.NewUnionPermissions(resolvedRoles))
	userInfo := &storage.UserInfo{}
	userInfo.SetUsername(externalUserClaim.UserID)
	userInfo.SetFriendlyName(externalUserClaim.FullName)
	userInfo.SetPermissions(ur)
	userInfo.SetRoles(utils.ExtractRolesForUserInfo(resolvedRoles))
	return userInfo
}
