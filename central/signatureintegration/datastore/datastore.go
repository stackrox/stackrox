package datastore

import (
	"context"

	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/signatureintegration/store"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore for signature integrations.
type DataStore interface {
	GetSignatureIntegration(ctx context.Context, id string) (*storage.SignatureIntegration, bool, error)
	GetAllSignatureIntegrations(ctx context.Context) ([]*storage.SignatureIntegration, error)
	CountSignatureIntegrations(ctx context.Context) (int, error)
	AddSignatureIntegration(ctx context.Context, integration *storage.SignatureIntegration) (*storage.SignatureIntegration, error)
	UpdateSignatureIntegration(ctx context.Context, integration *storage.SignatureIntegration) (bool, error)
	RemoveSignatureIntegration(ctx context.Context, id string) error
}

// New returns an instance of DataStore.
func New(storage store.SignatureIntegrationStore, policyStore policyDataStore.DataStore) DataStore {
	return &datastoreImpl{
		storage:     storage,
		policyStore: policyStore,
	}
}
