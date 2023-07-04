package datastore

import (
	"context"

	"github.com/stackrox/rox/central/events/datastore/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/uuid"
)

var _ DataStore = (*datastoreImpl)(nil)

type datastoreImpl struct {
	store store.Store
}

func (d *datastoreImpl) AddEvent(ctx context.Context, event *storage.Event) error {
	event.Id = uuid.NewV4().String()
	return d.store.Upsert(ctx, event)
}

func (d *datastoreImpl) GetEvents(ctx context.Context) ([]*storage.Event, error) {
	var events []*storage.Event
	walkFn := func() error {
		events = events[:0]
		return d.store.Walk(ctx, func(obj *storage.Event) error {
			events = append(events, obj)
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(walkFn); err != nil {
		return nil, err
	}
	return events, nil
}
