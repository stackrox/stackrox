package datastore

import (
	"context"
	"errors"

	"github.com/stackrox/rox/central/notifier/datastore/internal/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	notifierSAC = sac.ForResource(resources.Notifier)
)

type datastoreImpl struct {
	storage store.Store
}

func (b *datastoreImpl) GetNotifier(ctx context.Context, id string) (*storage.Notifier, bool, error) {
	if ok, err := notifierSAC.ReadAllowed(ctx); err != nil {
		return nil, false, err
	} else if !ok {
		return nil, false, nil
	}

	return b.storage.GetNotifier(id)
}

func (b *datastoreImpl) GetNotifiers(ctx context.Context, request *v1.GetNotifiersRequest) ([]*storage.Notifier, error) {
	if ok, err := notifierSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return b.storage.GetNotifiers(request)
}

func (b *datastoreImpl) AddNotifier(ctx context.Context, notifier *storage.Notifier) (string, error) {
	if ok, err := notifierSAC.WriteAllowed(ctx); err != nil {
		return "", err
	} else if !ok {
		return "", errors.New("permission denied")
	}

	return b.storage.AddNotifier(notifier)
}

func (b *datastoreImpl) UpdateNotifier(ctx context.Context, notifier *storage.Notifier) error {
	if ok, err := notifierSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return b.storage.UpdateNotifier(notifier)
}

func (b *datastoreImpl) RemoveNotifier(ctx context.Context, id string) error {
	if ok, err := notifierSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return b.storage.RemoveNotifier(id)
}
