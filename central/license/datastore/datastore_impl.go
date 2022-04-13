package datastore

import (
	"context"

	"github.com/stackrox/stackrox/central/license/internal/store"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/sac"
)

var (
	licenseSAC = sac.ForResource(resources.Licenses)
)

type dataStoreImpl struct {
	storage store.Store
}

func (ds *dataStoreImpl) ListLicenseKeys(ctx context.Context) ([]*storage.StoredLicenseKey, error) {
	if ok, err := licenseSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return ds.storage.ListLicenseKeys()
}

func (ds *dataStoreImpl) UpsertLicenseKeys(ctx context.Context, keys []*storage.StoredLicenseKey) error {
	if ok, err := licenseSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.storage.UpsertLicenseKeys(keys)
}

func (ds *dataStoreImpl) DeleteLicenseKey(ctx context.Context, licenseID string) error {
	if ok, err := licenseSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.storage.DeleteLicenseKey(licenseID)
}
