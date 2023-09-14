package datastore

import (
	"context"

	"github.com/pkg/errors"
	eventsSearch "github.com/stackrox/rox/central/administration/events/datastore/internal/search"
	"github.com/stackrox/rox/central/administration/events/datastore/internal/store"
	"github.com/stackrox/rox/central/administration/events/datastore/internal/writer"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
)

var (
	_        DataStore = (*datastoreImpl)(nil)
	eventSAC           = sac.ForResource(resources.Administration)
)

type datastoreImpl struct {
	searcher eventsSearch.Searcher
	store    store.Store
	// Buffered writer. Flush to initiate batched upsert.
	writer writer.Writer
}

func (ds *datastoreImpl) AddEvent(ctx context.Context, event *events.AdministrationEvent) error {
	// We need an explicit SAC check for AddEvent since writer will do a buffered write and first hold events
	// in-memory before flushing them to the database. Without the SAC check here, it'd be possible for unauthorized
	// callers to add events to the buffer, and let them be flushed by an authorized caller.
	if err := sac.VerifyAuthzOK(eventSAC.WriteAllowed(ctx)); err != nil {
		return err
	}

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

func (ds *datastoreImpl) GetEvent(ctx context.Context, id string) (*storage.AdministrationEvent, error) {
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

func (ds *datastoreImpl) Search(ctx context.Context, query *v1.Query) ([]search.Result, error) {
	return ds.searcher.Search(ctx, query)
}

func (ds *datastoreImpl) RemoveEvents(ctx context.Context, ids ...string) error {
	return ds.store.DeleteMany(ctx, ids)
}

func (ds *datastoreImpl) Flush(ctx context.Context) error {
	err := ds.writer.Flush(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to flush administration events")
	}
	return nil
}
