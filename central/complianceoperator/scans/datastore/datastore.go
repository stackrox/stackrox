package datastore

import (
	"context"

	"github.com/pkg/errors"
	store "github.com/stackrox/rox/central/complianceoperator/scans/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	complianceOperatorSAC = sac.ForResource(resources.ComplianceOperator)
)

// DataStore defines the possible interactions with compliance operator rules
type DataStore interface {
	Walk(ctx context.Context, fn func(rule *storage.ComplianceOperatorScan) error) error
	Upsert(ctx context.Context, rule *storage.ComplianceOperatorScan) error
	Delete(ctx context.Context, id string) error
}

// NewDatastore returns the datastore wrapper for compliance operator rules
func NewDatastore(store store.Store) DataStore {
	return &datastoreImpl{
		store: store,
	}
}

type datastoreImpl struct {
	store store.Store
}

func (d *datastoreImpl) Walk(ctx context.Context, fn func(scan *storage.ComplianceOperatorScan) error) error {
	if ok, err := complianceOperatorSAC.ReadAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.Wrap(sac.ErrResourceAccessDenied, "compliance operator scan read")
	}
	return d.store.Walk(ctx, fn)
}

func (d *datastoreImpl) Upsert(ctx context.Context, scan *storage.ComplianceOperatorScan) error {
	if ok, err := complianceOperatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.Wrap(sac.ErrResourceAccessDenied, "compliance operator scan write")
	}
	return d.store.Upsert(ctx, scan)
}

func (d *datastoreImpl) Delete(ctx context.Context, id string) error {
	if ok, err := complianceOperatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.Wrap(sac.ErrResourceAccessDenied, "compliance operator scan delete")
	}
	return d.store.Delete(ctx, id)
}
