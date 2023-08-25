package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/notifications/datastore/internal/search"
	"github.com/stackrox/rox/central/notifications/datastore/internal/store"
	"github.com/stackrox/rox/central/notifications/datastore/internal/writer"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
)

var _ DataStore = (*datastoreImpl)(nil)

type datastoreImpl struct {
	searcher search.Searcher
	store    store.Store
	writer   writer.Writer
}

func (ds *datastoreImpl) AddNotification(ctx context.Context, notification *storage.Notification) error {
	if err := ds.writer.Upsert(ctx, notification); err != nil {
		return errors.Wrap(err, "failed to upsert notification")
	}
	return nil
}

func (ds *datastoreImpl) CountNotifications(ctx context.Context, query *v1.Query) (int, error) {
	count, err := ds.searcher.Count(ctx, query)
	if err != nil {
		return 0, errors.Wrap(err, "failed to count notifications")
	}
	return count, nil
}

func (ds *datastoreImpl) GetNotificationByID(ctx context.Context, id string) (*storage.Notification, error) {
	notification, exists, err := ds.store.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get notification by id")
	}
	if !exists {
		return nil, errox.NotFound.Newf("notification %q not found", id)
	}
	return notification, nil
}

func (ds *datastoreImpl) ListNotifications(ctx context.Context, query *v1.Query) ([]*storage.Notification, error) {
	notifications, err := ds.store.GetByQuery(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get notifications by query")
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
