package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/notifier/datastore/internal/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/secrets"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	integrationSAC = sac.ForResource(resources.Integration)
)

type datastoreImpl struct {
	lock    sync.Mutex
	storage store.Store
}

func verifyOrigin(ctx context.Context, n *storage.Notifier) error {
	if !declarativeconfig.CanModifyResource(ctx, n) {
		return errox.NotAuthorized.Newf("notifier %q's origin is %s, "+
			"cannot be modified or deleted with the current permission",
			n.GetName(), n.GetTraits().GetOrigin())
	}
	return nil
}

func (b *datastoreImpl) verifyExistsAndMutable(ctx context.Context, id string, force bool) (*storage.Notifier, error) {
	notifier, exists, err := b.storage.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errox.NotFound.Newf("notifier with id %q was not found", id)
	}

	switch notifier.GetTraits().GetMutabilityMode() {
	case storage.Traits_ALLOW_MUTATE:
		return notifier, nil
	case storage.Traits_ALLOW_MUTATE_FORCED:
		if force {
			return notifier, nil
		}
		return nil, errox.InvalidArgs.Newf("notifier %q is immutable "+
			"and can only be removed via API and specifying the force flag", id)
	default:
		utils.Should(errox.InvalidArgs.Newf("unknown mutability mode given: %q",
			notifier.GetTraits().GetMutabilityMode()))
	}
	return nil, errox.InvalidArgs.Newf("notifier %q is immutable", id)
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

	if err := verifyOrigin(ctx, notifier); err != nil {
		return "", errors.Wrap(err, "origin didn't match for new notifier")
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

func (b *datastoreImpl) UpdateNotifier(ctx context.Context, notifier *storage.Notifier) error {
	if ok, err := integrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	existing, err := b.verifyExistsAndMutable(ctx, notifier.GetId(), false)
	if err != nil {
		return err
	}
	if err = verifyOrigin(ctx, existing); err != nil {
		return errors.Wrap(err, "origin didn't match for existing notifier")
	}
	if err = verifyOrigin(ctx, notifier); err != nil {
		return errors.Wrap(err, "origin didn't match for new notifier")
	}
	return b.storage.Upsert(ctx, notifier)
}

func (b *datastoreImpl) RemoveNotifier(ctx context.Context, id string) error {
	if ok, err := integrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	existing, err := b.verifyExistsAndMutable(ctx, id, false)
	if err != nil {
		return err
	}
	if err = verifyOrigin(ctx, existing); err != nil {
		return errors.Wrap(err, "origin didn't match for existing notifier")
	}
	return b.storage.Delete(ctx, id)
}
