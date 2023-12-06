package auth

import (
	securitycenter "cloud.google.com/go/securitycenter/apiv1"
	"cloud.google.com/go/storage"
	"github.com/stackrox/rox/pkg/cloudproviders/gcp/handler"
)

// STSClientManager manages GCP clients with short-lived credentials.
type STSClientManager interface {
	Start()
	Stop()

	StorageClientHandler() handler.Handler[*storage.Client]
	SecurityCenterClientHandler() handler.Handler[*securitycenter.Client]
}
