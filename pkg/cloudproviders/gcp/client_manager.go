package gcp

import (
	"context"

	"cloud.google.com/go/storage"
)

// STSClientManager manages GCP clients with short-lived credentials.
type STSClientManager interface {
	Start()
	Stop()

	StorageClient(ctx context.Context) (*storage.Client, func())
}
