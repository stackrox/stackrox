package sac

import (
	"context"

	"github.com/stackrox/default-authz-plugin/pkg/payload"
)

// ScopeRequest is the interface which must be implemented by anything tracked by ScopeRequestTracker
type ScopeRequest interface {
	GetAccessScope() payload.AccessScope
	SetState(state TryAllowedResult)
}

// ScopeRequestTracker tracks the list of pending permission checks
//go:generate mockgen-wrapper ScopeRequestTracker
type ScopeRequestTracker interface {
	AddRequested(scopes ...ScopeRequest)
	PerformChecks(ctx context.Context) error
}
