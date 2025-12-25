package tokens

import (
	"encoding/json"
	"time"

	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stackrox/rox/generated/storage"
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
	//
	// Deprecated: Use RoleNames instead.
	RoleName string `json:"role,omitempty"`
	// RoleNames represents the claim that the user identified by the token has the given roles.
	RoleNames []string `json:"roles,omitempty"`
	// ExternalUser represents the claim that this token identifies a user from an external identity provider.
	ExternalUser *ExternalUserClaim `json:"external_user,omitempty"`
	// Name represents the name of the token assigned by the creator.
	Name     string `json:"name,omitempty"`
	ExpireAt *time.Time
	// DynamicScope represents an ephemeral access scope embedded in the token claims.
	// Unlike regular access scopes (referenced by ID), dynamic scopes are never persisted
	// to the database and are used for short-lived, scoped tokens (e.g., from Sensor's
	// GraphQL gateway for OCP console plugin requests).
	DynamicScope *storage.DynamicAccessScope `json:"dynamic_scope,omitempty"`
}

// Claims are the claims contained in a token.
type Claims struct {
	jwt.Claims
	RoxClaims

	Extra map[string]json.RawMessage
}
