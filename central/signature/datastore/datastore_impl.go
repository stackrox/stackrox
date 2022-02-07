package datastore

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/signature/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	signatureSAC = sac.ForResource(resources.SignatureIntegration)
)

type datastoreImpl struct {
	storage store.SignatureIntegrationStore
}

func (d *datastoreImpl) GetSignatureIntegration(ctx context.Context, id string) (*storage.SignatureIntegration, bool, error) {
	if ok, err := signatureSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, false, err
	}

	return d.storage.Get(id)
}

func (d *datastoreImpl) GetSignatureIntegrations(ctx context.Context) ([]*storage.SignatureIntegration, error) {
	if ok, err := signatureSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, err
	}

	var integrations []*storage.SignatureIntegration
	err := d.storage.Walk(func(integration *storage.SignatureIntegration) error {
		integrations = append(integrations, integration)
		return nil
	})
	return integrations, err
}

func (d *datastoreImpl) AddSignatureIntegration(ctx context.Context, integration *storage.SignatureIntegration) error {
	if err := sac.VerifyAuthzOK(signatureSAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	_, found, err := d.storage.Get(integration.GetId())
	if err != nil {
		return err
	} else if found {
		return fmt.Errorf("signature integration id=%s already exists, requested name=%q", integration.GetId(), integration.GetName())
	}

	return d.storage.Upsert(integration)
}

func (d *datastoreImpl) UpdateSignatureIntegration(ctx context.Context, integration *storage.SignatureIntegration) error {
	if err := sac.VerifyAuthzOK(signatureSAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	_, found, err := d.storage.Get(integration.GetId())
	if err != nil {
		return err
	} else if !found {
		return fmt.Errorf("signature integration id=%s doesn't exist, requested name=%q", integration.GetId(), integration.GetName())
	}

	return d.storage.Upsert(integration)
}

func (d *datastoreImpl) RemoveSignatureIntegration(ctx context.Context, id string) error {
	if err := sac.VerifyAuthzOK(signatureSAC.WriteAllowed(ctx)); err != nil {
		return err
	}

	return d.storage.Delete(id)
}
