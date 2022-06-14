package datastore

import (
	"context"

	"github.com/stackrox/rox/central/notifier/datastore/internal/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/secrets"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	notifierSAC = sac.ForResource(resources.Notifier)
)

type datastoreImpl struct {
	lock    sync.Mutex
	storage store.Store
}

func (b *datastoreImpl) GetNotifier(ctx context.Context, id string) (*storage.Notifier, bool, error) {
	if ok, err := notifierSAC.ReadAllowed(ctx); err != nil {
		return nil, false, err
	} else if !ok {
		return nil, false, nil
	}

	return b.storage.Get(ctx, id)
}

func (b *datastoreImpl) GetScrubbedNotifier(ctx context.Context, id string) (*storage.Notifier, bool, error) {
	notifier, exists, err := b.GetNotifier(ctx, id)
	if err != nil || !exists {
		return notifier, exists, err
	}

	secrets.ScrubSecretsFromStructWithReplacement(notifier, secrets.ScrubReplacementStr)

	return notifier, exists, err
}

func (b *datastoreImpl) GetNotifiers(ctx context.Context) ([]*storage.Notifier, error) {
	if ok, err := notifierSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return b.storage.GetAll(ctx)
}

func (b *datastoreImpl) GetScrubbedNotifiers(ctx context.Context) ([]*storage.Notifier, error) {
	notifiers, err := b.GetNotifiers(ctx)
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
	notifier.Id = uuid.NewV4().String()

	b.lock.Lock()
	defer b.lock.Unlock()

	exists, err := b.storage.Exists(ctx, notifier.GetId())
	if err != nil {
		return notifier.GetId(), err
	}
	if exists {
		return notifier.GetId(), errox.InvalidArgs.Newf("notifier with id %q was found", notifier.GetId())
	}
	return notifier.GetId(), b.storage.Upsert(ctx, notifier)
}

func (b *datastoreImpl) UpdateNotifier(ctx context.Context, notifier *storage.Notifier) error {
	if ok, err := notifierSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	exists, err := b.storage.Exists(ctx, notifier.GetId())
	if err != nil {
		return err
	}
	if !exists {
		return errox.NotFound.Newf("notifier with id %q was not found", notifier.GetId())
	}

	return b.storage.Upsert(ctx, notifier)
}

func (b *datastoreImpl) RemoveNotifier(ctx context.Context, id string) error {
	if ok, err := notifierSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return b.storage.Delete(ctx, id)
}
