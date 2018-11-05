package tokens

import (
	"encoding/json"

	"github.com/stackrox/rox/generated/api/v1"
	"gopkg.in/square/go-jose.v2/jwt"
)

// ExternalUserClaim represents the claim that this token identifies a user from an external identity provider.
type ExternalUserClaim struct {
	UserID   string `json:"user_id"`
	FullName string `json:"full_name,omitempty"`
	Email    string `json:"email,omitempty"`
}

// RoxClaims are the claims used for authentication by the StackRox container security platform.
type RoxClaims struct {
	// Permissions represents the claim that the user identified by the token has the given permissions. Setting this
	// makes the token a pure by-value token.
	Permissions []*v1.Permission `json:"permissions,omitempty"`
	// Role represents the claim that the user identified by the token has the given role.
	RoleName string `json:"role,omitempty"`
	// ExternalUser represents the claim that this token identifies a user from an external identity provider.
	ExternalUser *ExternalUserClaim `json:"external_user,omitempty"`
}

// Claims are the claims contained in a token.
type Claims struct {
	jwt.Claims
	RoxClaims

	Extra map[string]json.RawMessage
}
