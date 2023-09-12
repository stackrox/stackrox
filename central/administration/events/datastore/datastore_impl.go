package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/administration/events/datastore/internal/search"
	"github.com/stackrox/rox/central/administration/events/datastore/internal/store"
	"github.com/stackrox/rox/central/administration/events/datastore/internal/writer"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/errox"
)

var _ DataStore = (*datastoreImpl)(nil)

type datastoreImpl struct {
	searcher search.Searcher
	store    store.Store
	// Buffered writer. Flush to initiate batched upsert.
	writer writer.Writer
}

func (ds *datastoreImpl) AddEvent(ctx context.Context, event *events.AdministrationEvent) error {
	// The writer handles the SAC checks for the event.
	if err := ds.writer.Upsert(ctx, event); err != nil {
		return errors.Wrap(err, "failed to upsert administration event")
	}
	return nil
}

func (ds *datastoreImpl) CountEvents(ctx context.Context, query *v1.Query) (int, error) {
	count, err := ds.searcher.Count(ctx, query)
	if err != nil {
		return 0, errors.Wrap(err, "failed to count administration events")
	}
	return count, nil
}

func (ds *datastoreImpl) GetEventByID(ctx context.Context, id string) (*storage.AdministrationEvent, error) {
	event, exists, err := ds.store.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get administration event")
	}
	if !exists {
		return nil, errox.NotFound.Newf("administration event %q not found", id)
	}
	return event, nil
}

func (ds *datastoreImpl) ListEvents(ctx context.Context, query *v1.Query) ([]*storage.AdministrationEvent, error) {
	events, err := ds.store.GetByQuery(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list administration events")
	}
	return events, nil
}

func (ds *datastoreImpl) Flush(ctx context.Context) error {
	err := ds.writer.Flush(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to flush administration events")
	}
	return nil
}
