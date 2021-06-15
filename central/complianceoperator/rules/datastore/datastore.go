package datastore

import (
	"context"
	"errors"

	store "github.com/stackrox/rox/central/complianceoperator/rules/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	complianceOperatorSAC = sac.ForResource(resources.ComplianceOperator)
)

// DataStore defines the possible interactions with compliance operator rules
type DataStore interface {
	Walk(ctx context.Context, fn func(rule *storage.ComplianceOperatorRule) error) error
	Upsert(ctx context.Context, rule *storage.ComplianceOperatorRule) error
	Delete(ctx context.Context, id string) error
}

// NewDatastore returns the datastore wrapper for compliance operator rules
func NewDatastore(store store.Store) DataStore {
	return &datastoreImpl{store: store}
}

type datastoreImpl struct {
	store store.Store
}

func (d *datastoreImpl) Walk(ctx context.Context, fn func(rule *storage.ComplianceOperatorRule) error) error {
	if ok, err := complianceOperatorSAC.ReadAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("read access denied for compliance operator rules")
	}
	return d.store.Walk(fn)
}

func (d *datastoreImpl) Upsert(ctx context.Context, rule *storage.ComplianceOperatorRule) error {
	if ok, err := complianceOperatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("write access denied for compliance operator rules")
	}
	return d.store.Upsert(rule)
}

func (d *datastoreImpl) Delete(ctx context.Context, id string) error {
	if ok, err := complianceOperatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("write access denied for compliance operator rules")
	}
	return d.store.Delete(id)
}
