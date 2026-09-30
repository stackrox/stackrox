package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/customresource/datastore/internal/store"
	pgStore "github.com/stackrox/rox/central/customresource/datastore/internal/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/search"
)

// DataStore provides access to tracked Kubernetes custom resources.
//
//go:generate mockgen-wrapper
type DataStore interface {
	GetCustomResource(ctx context.Context, id string) (*storage.CustomResource, bool, error)
	ListCustomResources(ctx context.Context, query *v1.Query) ([]*storage.CustomResource, error)
	Search(ctx context.Context, query *v1.Query) ([]search.Result, error)
	UpsertCustomResource(ctx context.Context, customResource *storage.CustomResource) error
	DeleteCustomResource(ctx context.Context, id string) error
	GetOwnedDeploymentIDs(ctx context.Context, id string) ([]string, error)
}

func newDataStore(store store.Store, db postgres.DB) DataStore {
	return &datastoreImpl{
		store: store,
		db:    db,
	}
}

// GetTestPostgresDataStore provides a datastore connected to Postgres for testing.
func GetTestPostgresDataStore(_ testing.TB, db postgres.DB) DataStore {
	return newDataStore(pgStore.New(db), db)
}
