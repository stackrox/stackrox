package gcp

import (
	"context"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

//go:generate mockgen-wrapper
type StorageClientFactory interface {
	NewClient(ctx context.Context, opts ...option.ClientOption) (*storage.Client, error)
}

type gcpStorageClientFactory struct{}

var _ StorageClientFactory = &gcpStorageClientFactory{}

func (g *gcpStorageClientFactory) NewClient(ctx context.Context, opts ...option.ClientOption) (*storage.Client, error) {
	return storage.NewClient(ctx, opts...)
}
