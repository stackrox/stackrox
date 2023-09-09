package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/administration/events/datastore/internal/search"
	"github.com/stackrox/rox/central/administration/events/datastore/internal/store"
	"github.com/stackrox/rox/central/administration/events/datastore/internal/writer"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
)

var _ DataStore = (*datastoreImpl)(nil)

type datastoreImpl struct {
	searcher search.Searcher
	store    store.Store
	// Buffered writer. Flush to initiate batched upsert.
	writer writer.Writer
}

// The writer handles the SAC checks for the event.
func (ds *datastoreImpl) AddEvent(ctx context.Context, event *storage.AdministrationEvent) error {
	if err := ds.writer.Upsert(ctx, event); err != nil {
		return errors.Wrap(err, "failed to upsert notification")
	}
	return nil
}

func (ds *datastoreImpl) CountEvents(ctx context.Context, query *v1.Query) (int, error) {
	count, err := ds.searcher.Count(ctx, query)
	if err != nil {
		return 0, errors.Wrap(err, "failed to count notifications")
	}
	return count, nil
}

func (ds *datastoreImpl) GetEventByID(ctx context.Context, id string) (*storage.AdministrationEvent, error) {
	notification, exists, err := ds.store.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get notification")
	}
	if !exists {
		return nil, errox.NotFound.Newf("notification %q not found", id)
	}
	return notification, nil
}

func (ds *datastoreImpl) ListEvents(ctx context.Context, query *v1.Query) ([]*storage.AdministrationEvent, error) {
	notifications, err := ds.store.GetByQuery(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list notifications")
	}
	return notifications, nil
}

func (ds *datastoreImpl) Flush(ctx context.Context) error {
	err := ds.writer.Flush(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to flush notifications")
	}
	return nil
}
