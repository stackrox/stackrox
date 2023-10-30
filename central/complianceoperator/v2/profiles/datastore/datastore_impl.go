package datastore

import (
	"context"

	"github.com/stackrox/rox/central/complianceoperator/v2/profiles/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
)

var (
	complianceOperatorSAC = sac.ForResource(resources.ComplianceOperator)
)

type datastoreImpl struct {
	store postgres.Store
}

// UpsertProfile adds the profile to the database
func (d *datastoreImpl) UpsertProfile(ctx context.Context, profile *storage.ComplianceOperatorProfileV2) error {
	if ok, err := complianceOperatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	// See if profile already exists
	profiles, err := d.store.GetByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorProfileName, profile.GetName()).
		AddExactMatches(search.ComplianceOperatorProfileVersion).ProtoQuery())
	if err != nil {
		return err
	}

	// We already have this profile, so move along.
	if len(profiles) > 0 {
		return nil
	}

	return d.store.Upsert(ctx, profile)
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
