package datastore

import (
	"context"

	"github.com/stackrox/rox/central/complianceoperator/v2/profiles/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
)

type datastoreImpl struct {
	store postgres.Store
}

// UpsertProfile adds the profile to the database
func (d *datastoreImpl) UpsertProfile(_ context.Context, _ *storage.ComplianceOperatorProfileV2) error {
	return errox.NotImplemented
}

// DeleteProfile removes a profile from the database
func (d *datastoreImpl) DeleteProfile(_ context.Context, _ string) error {
	return errox.NotImplemented
}
