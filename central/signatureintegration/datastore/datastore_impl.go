package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/signatureintegration/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	signatureSAC = sac.ForResource(resources.SignatureIntegration)
)

type datastoreImpl struct {
	storage store.SignatureIntegrationStore

	lock sync.RWMutex
}

func (d *datastoreImpl) GetSignatureIntegration(ctx context.Context, id string) (*storage.SignatureIntegration, bool, error) {
	if ok, err := signatureSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, false, err
	}

	return d.storage.Get(ctx, id)
}

func (d *datastoreImpl) GetAllSignatureIntegrations(ctx context.Context) ([]*storage.SignatureIntegration, error) {
	if ok, err := signatureSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, err
	}

	var integrations []*storage.SignatureIntegration
	err := d.storage.Walk(ctx, func(integration *storage.SignatureIntegration) error {
		integrations = append(integrations, integration)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return integrations, nil
}

func (d *datastoreImpl) AddSignatureIntegration(ctx context.Context, integration *storage.SignatureIntegration) (*storage.SignatureIntegration, error) {
	if err := sac.VerifyAuthzOK(signatureSAC.WriteAllowed(ctx)); err != nil {
		return nil, err
	}
	if integration.GetId() != "" {
		return nil, errox.InvalidArgs.Newf("id should be empty but %q provided", integration.GetId())
	}
	integration.Id = GenerateSignatureIntegrationID()
	if err := ValidateSignatureIntegration(integration); err != nil {
		return nil, errox.NewErrInvalidArgs(err.Error())
	}

	// Protect against TOCTOU race condition.
	d.lock.Lock()
	defer d.lock.Unlock()

	if err := d.verifyIntegrationIDDoesNotExist(ctx, integration.GetId()); err != nil {
		if errors.Is(err, errox.AlreadyExists) {
			return nil, errors.Wrap(err, "collision in generated signature integration id, try again")
		}
		return nil, err
	}

	err := d.storage.Upsert(ctx, integration)
	if err != nil {
		return nil, err
	}
	return integration, nil
}

func (d *datastoreImpl) UpdateSignatureIntegration(ctx context.Context, integration *storage.SignatureIntegration) (bool, error) {
	if err := sac.VerifyAuthzOK(signatureSAC.WriteAllowed(ctx)); err != nil {
		return false, err
	}
	if err := ValidateSignatureIntegration(integration); err != nil {
		return false, errox.NewErrInvalidArgs(err.Error())
	}

	// Protect against TOCTOU race condition.
	d.lock.Lock()
	defer d.lock.Unlock()

	hasUpdatedPublicKeys, err := d.verifyIntegrationIDAndUpdates(ctx, integration)
	if err != nil {
		return false, err
	}

	return hasUpdatedPublicKeys, d.storage.Upsert(ctx, integration)
}

func (d *datastoreImpl) RemoveSignatureIntegration(ctx context.Context, id string) error {
	if err := sac.VerifyAuthzOK(signatureSAC.WriteAllowed(ctx)); err != nil {
		return err
	}

	d.lock.Lock()
	defer d.lock.Unlock()

	if err := d.verifyIntegrationIDExists(ctx, id); err != nil {
		return err
	}

	return d.storage.Delete(ctx, id)
}

func (d *datastoreImpl) verifyIntegrationIDExists(ctx context.Context, id string) error {
	_, err := d.getSignatureIntegrationByID(ctx, id)
	if err != nil {
		return err
	}
	return nil
}

func (d *datastoreImpl) getSignatureIntegrationByID(ctx context.Context, id string) (*storage.SignatureIntegration, error) {
	integration, found, err := d.storage.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if !found {
		return nil, errox.NotFound.Newf("signature integration id=%s doesn't exist", id)
	}
	return integration, nil
}

func (d *datastoreImpl) verifyIntegrationIDAndUpdates(ctx context.Context,
	updatedIntegration *storage.SignatureIntegration) (bool, error) {
	existingIntegration, err := d.getSignatureIntegrationByID(ctx, updatedIntegration.GetId())
	if err != nil {
		return false, err
	}
	return !getPublicKeyPEMSet(existingIntegration).Equal(getPublicKeyPEMSet(updatedIntegration)), nil
}

func (d *datastoreImpl) verifyIntegrationIDDoesNotExist(ctx context.Context, id string) error {
	_, found, err := d.storage.Get(ctx, id)
	if err != nil {
		return err
	} else if found {
		return errox.AlreadyExists.Newf("signature integration id=%s already exists", id)
	}
	return nil
}

func getPublicKeyPEMSet(integration *storage.SignatureIntegration) set.StringSet {
	publicKeySet := set.NewStringSet()
	for _, key := range integration.GetCosign().GetPublicKeys() {
		publicKeySet.Add(key.GetPublicKeyPemEnc())
	}
	return publicKeySet
}
