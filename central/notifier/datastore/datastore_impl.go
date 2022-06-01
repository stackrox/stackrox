package datastore

import (
	"context"

	"github.com/stackrox/rox/central/notifier/datastore/internal/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/secrets"
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

func (b *datastoreImpl) GetScrubbedNotifier(ctx context.Context, id string) (*storage.Notifier, bool, error) {
	notifier, exists, err := b.GetNotifier(ctx, id)
	if err != nil || !exists {
		return notifier, exists, err
	}

	secrets.ScrubSecretsFromStructWithReplacement(notifier, secrets.ScrubReplacementStr)

	return notifier, exists, err
}

func (b *datastoreImpl) GetNotifiers(ctx context.Context, request *v1.GetNotifiersRequest) ([]*storage.Notifier, error) {
	if ok, err := notifierSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return b.storage.GetNotifiers(request)
}

func (b *datastoreImpl) GetScrubbedNotifiers(ctx context.Context, request *v1.GetNotifiersRequest) ([]*storage.Notifier, error) {
	notifiers, err := b.GetNotifiers(ctx, request)
	if err != nil {
		return nil, err
	}

	for _, notifier := range notifiers {
		secrets.ScrubSecretsFromStructWithReplacement(notifier, secrets.ScrubReplacementStr)
	}

	return notifiers, nil
}

func (b *datastoreImpl) AddNotifier(ctx context.Context, notifier *storage.Notifier) (string, error) {
	if ok, err := notifierSAC.WriteAllowed(ctx); err != nil {
		return "", err
	} else if !ok {
		return "", sac.ErrResourceAccessDenied
	}

	return b.storage.AddNotifier(notifier)
}

func (b *datastoreImpl) UpdateNotifier(ctx context.Context, notifier *storage.Notifier) error {
	if ok, err := notifierSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return b.storage.UpdateNotifier(notifier)
}

func (b *datastoreImpl) RemoveNotifier(ctx context.Context, id string) error {
	if ok, err := notifierSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return b.storage.RemoveNotifier(id)
}
