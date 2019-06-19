package client

import (
	"context"
	"net/http"

	"github.com/stackrox/default-authz-plugin/pkg/payload"
	"github.com/stackrox/rox/generated/storage"
)

// Client is a simple interface describing retrieving some per user data from a separate service.
//go:generate mockgen-wrapper Client
type Client interface {
	ForUser(ctx context.Context, principal payload.Principal, scopes ...payload.AccessScope) (allowed, denied []payload.AccessScope, err error)
}

// New returns a new instance of Client.
func New(config *storage.AuthzPluginConfig) Client {
	return &clientImpl{
		client:       http.DefaultClient,
		authEndpoint: config.GetEndpointConfig().GetEndpoint(),
	}
}
