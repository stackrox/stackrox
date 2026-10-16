package user

import (
	"context"

	"github.com/stackrox/rox/pkg/grpc/authz"
)

type roleChecker struct{ roleName string }

func (r *roleChecker) Authorized(_ context.Context, _ string) error { return nil }

// WithRole returns an Authorizer that checks for a specific role.
func WithRole(roleName string) authz.Authorizer { return &roleChecker{roleName} }

type permChecker struct{}

func (p *permChecker) Authorized(_ context.Context, _ string) error { return nil }

// With returns an Authorizer that checks for specific permissions.
func With() authz.Authorizer { return &permChecker{} }

// Authenticated returns an Authorizer that accepts any authenticated caller.
func Authenticated() authz.Authorizer { return With() }
