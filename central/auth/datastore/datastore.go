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
	ForEachAuthM2MConfig(ctx context.Context, fn func(obj *storage.AuthMachineToMachineConfig) error) error
	UpsertAuthM2MConfig(ctx context.Context, config *storage.AuthMachineToMachineConfig) (*storage.AuthMachineToMachineConfig, error)
	RemoveAuthM2MConfig(ctx context.Context, id string) error

	GetTokenExchanger(ctx context.Context, issuer string) (m2m.TokenExchanger, bool)
	InitializeTokenExchangers() error
}

// New returns an instance of an auth machine to machine Datastore.
func New(store store.Store, set m2m.TokenExchangerSet, issuerFetcher m2m.ServiceAccountIssuerFetcher) DataStore {
	return &datastoreImpl{
		store:         store,
		set:           set,
		issuerFetcher: issuerFetcher,
	}
}
