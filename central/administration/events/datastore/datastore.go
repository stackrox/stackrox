package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/administration/events/datastore/internal/search"
	"github.com/stackrox/rox/central/administration/events/datastore/internal/store"
	pgStore "github.com/stackrox/rox/central/administration/events/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/administration/events/datastore/internal/writer"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/postgres"
)

// DataStore provides an interface to handle administration events.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// AddEvent is intended to be used by the administration event handler
	// to add events to the buffered writer.
	AddEvent(ctx context.Context, event *events.AdministrationEvent) error
	// Flush initiates a batched upsert to the database.
	Flush(ctx context.Context) error

	CountEvents(ctx context.Context, query *v1.Query) (int, error)
	GetEvent(ctx context.Context, id string) (*storage.AdministrationEvent, error)
	ListEvents(ctx context.Context, query *v1.Query) ([]*storage.AdministrationEvent, error)
}

func newDataStore(searcher search.Searcher, storage store.Store, writer writer.Writer) DataStore {
	return &datastoreImpl{
		searcher: searcher,
		store:    storage,
		writer:   writer,
	}
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) DataStore {
	searcher := search.New(pgStore.NewIndexer(pool))
	store := pgStore.New(pool)
	writer := writer.New(store)
	return newDataStore(searcher, store, writer)
}

// UpsertTestEvents provides a way to upsert storage.AdministrationEvents directly to the database.
// This is required for testing with custom timestamps, since the datastore expects a struct with only a subset
// of fields that clients may set. We still want this to be the case for callers, however for testing we can
// be more lax in our enforcement.
func UpsertTestEvents(ctx context.Context, _ testing.TB, datastore DataStore,
	events ...*storage.AdministrationEvent) error {
	return datastore.(*datastoreImpl).store.UpsertMany(ctx, events)
}

// RemoveTestEvents provides a way to remove storage.AdministrationEvents directly from the database.
// This is required for test cleanups, since the datastore doesn't expose functionality to remove events for clients.
func RemoveTestEvents(ctx context.Context, _ testing.TB, datastore DataStore,
	ids ...string) error {
	return datastore.(*datastoreImpl).store.DeleteMany(ctx, ids)
}
