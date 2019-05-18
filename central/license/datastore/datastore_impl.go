package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/license/internal/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
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
		return errors.New("permission denied")
	}

	return ds.storage.UpsertLicenseKeys(keys)
}

func (ds *dataStoreImpl) DeleteLicenseKey(ctx context.Context, licenseID string) error {
	if ok, err := licenseSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return ds.storage.DeleteLicenseKey(licenseID)
}
