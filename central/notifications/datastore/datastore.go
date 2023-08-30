package datastore

import (
	"context"

	"github.com/stackrox/rox/central/notifications/datastore/internal/search"
	"github.com/stackrox/rox/central/notifications/datastore/internal/store"
	"github.com/stackrox/rox/central/notifications/datastore/internal/writer"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore provides an interface to handle notifications.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// AddNotification is intended to be used by the notification handler to add
	// notifications to the buffered writer. Flush initiates a batched upsert to
	// the database.
	AddNotification(ctx context.Context, notification *storage.Notification) error
	Flush(ctx context.Context) error

	// These provide read-only access and are intended to be used by the API service.
	CountNotifications(ctx context.Context, query *v1.Query) (int, error)
	GetNotificationByID(ctx context.Context, id string) (*storage.Notification, error)
	ListNotifications(ctx context.Context, query *v1.Query) ([]*storage.Notification, error)
}

func newDataStore(searcher search.Searcher, storage store.Store, writer writer.Writer) DataStore {
	return &datastoreImpl{
		searcher: searcher,
		store:    storage,
		writer:   writer,
	}
}
