package gcp

import (
	"context"
	"net/http"

	"cloud.google.com/go/storage"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	googleHTTP "google.golang.org/api/transport/http"
)

// StorageClientFactory creates a GCP storage client.
//
//go:generate mockgen-wrapper
type StorageClientFactory interface {
	NewClient(ctx context.Context, creds *google.Credentials) (*storage.Client, error)
}

type gcpStorageClientFactory struct{}

var _ StorageClientFactory = &gcpStorageClientFactory{}

func (g *gcpStorageClientFactory) NewClient(ctx context.Context, creds *google.Credentials) (*storage.Client, error) {
	transport, err := googleHTTP.NewTransport(ctx, proxy.RoundTripper(), option.WithCredentials(creds))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create transport")
	}
	return storage.NewClient(ctx, option.WithHTTPClient(&http.Client{Transport: transport}))
}
