package datastore

import (
	"context"

	"github.com/stackrox/rox/central/auth/m2m"
	"github.com/stackrox/rox/central/auth/store"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore for auth machine to machine configs.
type DataStore interface {
	GetAuthM2MConfig(ctx context.Context, id string) (*storage.AuthMachineToMachineConfig, bool, error)
	ListAuthM2MConfigs(ctx context.Context) ([]*storage.AuthMachineToMachineConfig, error)
	AddAuthM2MConfig(ctx context.Context, config *storage.AuthMachineToMachineConfig) (*storage.AuthMachineToMachineConfig, error)
	UpdateAuthM2MConfig(ctx context.Context, config *storage.AuthMachineToMachineConfig) error
	RemoveAuthM2MConfig(ctx context.Context, id string) error

	GetTokenExchanger(ctx context.Context, issuer string) (m2m.TokenExchanger, bool)
}

// New returns an instance of an auth machine to machine Datastore.
func New(store store.Store, set m2m.TokenExchangerSet) DataStore {
	return &datastoreImpl{store: store, set: set}
}
