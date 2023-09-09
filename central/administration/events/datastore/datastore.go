package datastore

import (
	"context"

	"github.com/stackrox/rox/central/administration/events/datastore/internal/search"
	"github.com/stackrox/rox/central/administration/events/datastore/internal/store"
	"github.com/stackrox/rox/central/administration/events/datastore/internal/writer"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore provides an interface to handle notifications.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// AddEvent is intended to be used by the notification handler to add
	// notifications to the buffered writer.
	AddEvent(ctx context.Context, event *storage.AdministrationEvent) error
	// Flush initiates a batched upsert to the database.
	Flush(ctx context.Context) error

	CountEvents(ctx context.Context, query *v1.Query) (int, error)
	GetEventByID(ctx context.Context, id string) (*storage.AdministrationEvent, error)
	ListEvents(ctx context.Context, query *v1.Query) ([]*storage.AdministrationEvent, error)
}

func newDataStore(searcher search.Searcher, storage store.Store, writer writer.Writer) DataStore {
	return &datastoreImpl{
		searcher: searcher,
		store:    storage,
		writer:   writer,
	}
}
