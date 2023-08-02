package datastore

import (
	"context"

	"github.com/stackrox/rox/central/notifier/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore provides storage functionality for notifiers.
//
//go:generate mockgen-wrapper
type DataStore interface {
	GetNotifier(ctx context.Context, id string) (*storage.Notifier, bool, error)
	GetScrubbedNotifier(ctx context.Context, id string) (*storage.Notifier, bool, error)
	GetNotifiersFiltered(ctx context.Context, filter func(notifier *storage.Notifier) bool) ([]*storage.Notifier, error)
	GetNotifiers(ctx context.Context) ([]*storage.Notifier, error)
	GetManyNotifiers(ctx context.Context, notifierIDs []string) ([]*storage.Notifier, error)
	GetScrubbedNotifiers(ctx context.Context) ([]*storage.Notifier, error)
	Exists(ctx context.Context, id string) (bool, error)
	AddNotifier(ctx context.Context, notifier *storage.Notifier) (string, error)
	UpdateNotifier(ctx context.Context, notifier *storage.Notifier) error
	UpsertNotifier(ctx context.Context, notifier *storage.Notifier) (string, error)
	RemoveNotifier(ctx context.Context, id string) error
}

// New returns a new Store instance
func New(storage store.Store) DataStore {
	return &datastoreImpl{
		storage: storage,
	}
}
