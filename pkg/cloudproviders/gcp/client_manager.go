package gcp

import (
	"cloud.google.com/go/storage"
)

// STSClientManager manages GCP clients with short-lived credentials.
type STSClientManager interface {
	Start()
	Stop()

	StorageClient() (*storage.Client, func())
}
