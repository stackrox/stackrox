package client

import (
	"context"

	"github.com/stackrox/default-authz-plugin/pkg/payload"
)

// Client is a simple interface describing retrieving some per user data from a separate service.
type Client interface {
	ForUser(ctx context.Context, principal payload.Principal, scopes ...payload.AccessScope) (allowed, denied []payload.AccessScope, err error)
}
