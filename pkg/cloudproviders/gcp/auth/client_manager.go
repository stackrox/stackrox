package auth

import "github.com/stackrox/rox/pkg/cloudproviders/gcp/storage"

// STSClientManager manages GCP clients with short-lived credentials.
type STSClientManager interface {
	Start()
	Stop()

	StorageClientHandler() storage.ClientHandler
}
