package storage

import (
	"context"

	"cloud.google.com/go/storage"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

// ClientFactory creates a GCP storage client.
//
//go:generate mockgen-wrapper
type ClientFactory interface {
	NewClient(ctx context.Context, creds *google.Credentials) (*storage.Client, error)
}

var _ ClientFactory = &clientFactoryImpl{}

type clientFactoryImpl struct{}

func (s *clientFactoryImpl) NewClient(ctx context.Context, creds *google.Credentials) (*storage.Client, error) {
	return storage.NewClient(ctx, option.WithCredentials(creds))
}
