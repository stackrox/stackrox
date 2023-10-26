package datastore

import (
	"context"

	"github.com/stackrox/rox/central/complianceoperator/v2/profiles/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	complianceOperatorSAC = sac.ForResource(resources.ComplianceOperator)
)

type datastoreImpl struct {
	store postgres.Store
}

// UpsertProfile adds the profile to the database
func (d *datastoreImpl) UpsertProfile(ctx context.Context, result *storage.ComplianceOperatorProfileV2) error {
	if ok, err := complianceOperatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return d.store.Upsert(ctx, result)
}

// DeleteProfile removes a profile from the database
func (d *datastoreImpl) DeleteProfile(ctx context.Context, id string) error {
	if ok, err := complianceOperatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return d.store.Delete(ctx, id)
}
