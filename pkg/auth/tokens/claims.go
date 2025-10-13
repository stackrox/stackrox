package tokens

import (
	"encoding/json"
	"time"

	"github.com/go-jose/go-jose/v4/jwt"
)

// ExternalUserClaim represents the claim that this token identifies a user from an external identity provider.
type ExternalUserClaim struct {
	UserID   string `json:"user_id"`
	FullName string `json:"full_name,omitempty"`
	Email    string `json:"email,omitempty"`

	// Any extra information we want to attach.
	Attributes map[string][]string
}

// RoxClaims are the claims used for authentication by the StackRox Kubernetes security platform.
type RoxClaims struct {
	// Role represents the claim that the user identified by the token has the given role.
	// Deprecated: Use RoleNames instead.
	RoleName string `json:"role,omitempty"`
	// RoleNames represents the claim that the user identified by the token has the given roles.
	RoleNames []string `json:"roles,omitempty"`
	// ExternalUser represents the claim that this token identifies a user from an external identity provider.
	ExternalUser *ExternalUserClaim `json:"external_user,omitempty"`
	// Name represents the name of the token assigned by the creator.
	Name     string `json:"name,omitempty"`
	ExpireAt *time.Time
}

// Claims are the claims contained in a token.
type Claims struct {
	jwt.Claims
	RoxClaims

	Extra map[string]json.RawMessage
}
