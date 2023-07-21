package datastore

import (
	"context"

	"github.com/pkg/errors"
	store "github.com/stackrox/rox/central/complianceoperator/scansettingbinding/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	complianceOperatorSAC = sac.ForResource(resources.ComplianceOperator)
)

// DataStore defines the possible interactions with compliance operator scan setting bindings
type DataStore interface {
	Walk(ctx context.Context, fn func(result *storage.ComplianceOperatorScanSettingBinding) error) error
	Upsert(ctx context.Context, binding *storage.ComplianceOperatorScanSettingBinding) error
	Delete(ctx context.Context, id string) error
}

// NewDatastore returns the datastore wrapper for compliance operator scan setting bindings
func NewDatastore(store store.Store) DataStore {
	return &datastoreImpl{store: store}
}

type datastoreImpl struct {
	store store.Store
}

func (d *datastoreImpl) Walk(ctx context.Context, fn func(binding *storage.ComplianceOperatorScanSettingBinding) error) error {
	if ok, err := complianceOperatorSAC.ReadAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.Wrap(sac.ErrResourceAccessDenied, "compliance operator scan setting bindings read")
	}
	// Postgres retry in caller.
	return d.store.Walk(ctx, fn)
}

func (d *datastoreImpl) Upsert(ctx context.Context, binding *storage.ComplianceOperatorScanSettingBinding) error {
	if ok, err := complianceOperatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.Wrap(sac.ErrResourceAccessDenied, "compliance operator scan setting bindings write")
	}
	return d.store.Upsert(ctx, binding)
}

func (d *datastoreImpl) Delete(ctx context.Context, id string) error {
	if ok, err := complianceOperatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.Wrap(sac.ErrResourceAccessDenied, "compliance operator scan setting bindings write")
	}
	return d.store.Delete(ctx, id)
}
