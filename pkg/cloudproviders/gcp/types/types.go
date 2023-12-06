package types

import (
	securitycenter "cloud.google.com/go/securitycenter/apiv1"
	"cloud.google.com/go/storage"
)

// DoneFunc should be called to after work is done to release internally held locks.
type DoneFunc func()

// GcpSDKClients is the type constraints for all supported GCP SDK clients.
type GcpSDKClients interface {
	*storage.Client | *securitycenter.Client
}
