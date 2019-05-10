package datastore

import (
	"context"

	"github.com/stackrox/rox/central/notifier/datastore/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

type datastoreImpl struct {
	storage store.Store
}

func (b *datastoreImpl) GetNotifier(_ context.Context, id string) (*storage.Notifier, bool, error) {
	return b.storage.GetNotifier(id)
}

func (b *datastoreImpl) GetNotifiers(_ context.Context, request *v1.GetNotifiersRequest) ([]*storage.Notifier, error) {
	return b.storage.GetNotifiers(request)
}

func (b *datastoreImpl) AddNotifier(_ context.Context, notifier *storage.Notifier) (string, error) {
	return b.storage.AddNotifier(notifier)
}

func (b *datastoreImpl) UpdateNotifier(_ context.Context, notifier *storage.Notifier) error {
	return b.storage.UpdateNotifier(notifier)
}

func (b *datastoreImpl) RemoveNotifier(_ context.Context, id string) error {
	return b.storage.RemoveNotifier(id)
}
