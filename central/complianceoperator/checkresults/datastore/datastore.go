package datastore

import (
	"context"

	"github.com/pkg/errors"
	store "github.com/stackrox/stackrox/central/complianceoperator/checkresults/store"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/sac"
)

var (
	complianceOperatorSAC = sac.ForResource(resources.ComplianceOperator)
)

// DataStore defines the possible interactions with compliance operator check results
type DataStore interface {
	Walk(ctx context.Context, fn func(result *storage.ComplianceOperatorCheckResult) error) error
	Upsert(ctx context.Context, result *storage.ComplianceOperatorCheckResult) error
	Delete(ctx context.Context, id string) error
}

// NewDatastore returns the datastore wrapper for compliance operator check results
func NewDatastore(store store.Store) DataStore {
	return &datastoreImpl{store: store}
}

type datastoreImpl struct {
	store store.Store
}

func (d *datastoreImpl) Walk(ctx context.Context, fn func(result *storage.ComplianceOperatorCheckResult) error) error {
	if ok, err := complianceOperatorSAC.ReadAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.Wrap(sac.ErrResourceAccessDenied, "compliance operator check results read")
	}
	return d.store.Walk(ctx, fn)
}

func (d *datastoreImpl) Upsert(ctx context.Context, result *storage.ComplianceOperatorCheckResult) error {
	if ok, err := complianceOperatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.Wrap(sac.ErrResourceAccessDenied, "compliance operator check results write")
	}
	return d.store.Upsert(ctx, result)
}

func (d *datastoreImpl) Delete(ctx context.Context, id string) error {
	if ok, err := complianceOperatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.Wrap(sac.ErrResourceAccessDenied, "compliance operator check results write")
	}
	return d.store.Delete(ctx, id)
}
