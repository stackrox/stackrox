package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/notifier/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/secrets"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	integrationSAC            = sac.ForResource(resources.Integration)
	errMutabilityNotSupported = errox.InvalidArgs.New("notifiers do not support mutability mode")
)

type datastoreImpl struct {
	lock    sync.Mutex
	storage store.Store
}

func (b *datastoreImpl) UpsertNotifier(ctx context.Context, notifier *storage.Notifier) (string, error) {
	if err := sac.VerifyAuthzOK(integrationSAC.WriteAllowed(ctx)); err != nil {
		return "", err
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	existing, exists, err := b.GetNotifier(ctx, notifier.GetId())
	if err != nil {
		return "", err
	}
	if exists {
		if err = verifyNotifierOrigin(ctx, existing); err != nil {
			return "", errors.Wrap(err, "origin didn't match for existing notifier")
		}
	}
	if err = verifyNotifierOrigin(ctx, notifier); err != nil {
		return "", errors.Wrap(err, "origin didn't match for new notifier")
	}

	return notifier.GetId(), b.storage.Upsert(ctx, notifier)
}

func verifyNotifierOrigin(ctx context.Context, n *storage.Notifier) error {
	if !declarativeconfig.CanModifyResource(ctx, n) {
		return errox.NotAuthorized.Newf("notifier %q's origin is %s, "+
			"cannot be modified or deleted with the current permission",
			n.GetName(), n.GetTraits().GetOrigin())
	}
	return nil
}

func (b *datastoreImpl) GetNotifiersFiltered(ctx context.Context, filter func(notifier *storage.Notifier) bool) ([]*storage.Notifier, error) {
	if err := sac.VerifyAuthzOK(integrationSAC.ReadAllowed(ctx)); err != nil {
		return nil, err
	}
	// TODO: ROX-16071 add ability to pass filter to storage
	notifiers, err := b.storage.GetAll(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "getting notifiers from storage")
	}
	result := make([]*storage.Notifier, 0, len(notifiers))
	for _, n := range notifiers {
		if filter(n) {
			result = append(result, n)
		}
	}
	return result, nil
}

func (b *datastoreImpl) GetNotifier(ctx context.Context, id string) (*storage.Notifier, bool, error) {
	if ok, err := integrationSAC.ReadAllowed(ctx); err != nil {
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
	if ok, err := integrationSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return b.storage.GetAll(ctx)
}

func (b *datastoreImpl) GetManyNotifiers(ctx context.Context, notifierIDs []string) ([]*storage.Notifier, error) {
	notifiers, _, err := b.storage.GetMany(ctx, notifierIDs)
	if err != nil {
		return nil, err
	}
	return notifiers, nil
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

func (b *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	if ok, err := integrationSAC.ReadAllowed(ctx); err != nil {
		return false, err
	} else if !ok {
		return false, nil
	}

	found, err := b.storage.Exists(ctx, id)
	if err != nil || !found {
		return false, err
	}
	return true, nil
}

func (b *datastoreImpl) AddNotifier(ctx context.Context, notifier *storage.Notifier) (string, error) {
	if ok, err := integrationSAC.WriteAllowed(ctx); err != nil {
		return "", err
	} else if !ok {
		return "", sac.ErrResourceAccessDenied
	}
	notifier.Id = uuid.NewV4().String()

	if err := verifyNotifierOrigin(ctx, notifier); err != nil {
		return "", errors.Wrap(err, "origin didn't match for new notifier")
	}

	if notifier.GetTraits().GetMutabilityMode() != (*storage.Traits)(nil).GetMutabilityMode() {
		return "", errMutabilityNotSupported
	}

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

func (b *datastoreImpl) verifyExists(ctx context.Context, id string) (*storage.Notifier, error) {
	notifier, exists, err := b.GetNotifier(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errox.NotFound.Newf("notifier with id %q was not found", id)
	}
	return notifier, nil
}

func (b *datastoreImpl) UpdateNotifier(ctx context.Context, notifier *storage.Notifier) error {
	if ok, err := integrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	existing, err := b.verifyExists(ctx, notifier.GetId())
	if err != nil {
		return err
	}
	if err = verifyNotifierOrigin(ctx, existing); err != nil {
		return errors.Wrap(err, "origin didn't match for existing notifier")
	}
	if err = verifyNotifierOrigin(ctx, notifier); err != nil {
		return errors.Wrap(err, "origin didn't match for new notifier")
	}
	if notifier.GetTraits().GetMutabilityMode() != (*storage.Traits)(nil).GetMutabilityMode() {
		return errMutabilityNotSupported
	}
	return b.storage.Upsert(ctx, notifier)
}

func (b *datastoreImpl) RemoveNotifier(ctx context.Context, id string) error {
	if ok, err := integrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	existing, err := b.verifyExists(ctx, id)
	if err != nil {
		return err
	}
	if err = verifyNotifierOrigin(ctx, existing); err != nil {
		return errors.Wrap(err, "origin didn't match for existing notifier")
	}
	return b.storage.Delete(ctx, id)
}
