package datastore

import (
	"context"

	"github.com/pkg/errors"
	store "github.com/stackrox/rox/central/complianceoperator/rules/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	complianceOperatorSAC = sac.ForResource(resources.ComplianceOperator)
)

// DataStore defines the possible interactions with compliance operator rules
type DataStore interface {
	Walk(ctx context.Context, fn func(rule *storage.ComplianceOperatorRule) error) error
	Upsert(ctx context.Context, rule *storage.ComplianceOperatorRule) error
	Delete(ctx context.Context, id string) error
	GetRulesByName(ctx context.Context, name string) ([]*storage.ComplianceOperatorRule, error)
	ExistsByName(ctx context.Context, name string) (bool, error)
}

// NewDatastore returns the datastore wrapper for compliance operator rules
func NewDatastore(store store.Store) (DataStore, error) {
	ds := &datastoreImpl{
		store:       store,
		rulesByName: make(map[string]map[string]*storage.ComplianceOperatorRule),
	}
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.ComplianceOperator),
		))

	err := store.Walk(ctx, func(rule *storage.ComplianceOperatorRule) error {
		ds.addToRulesByNameNoLock(rule)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ds, nil
}

type datastoreImpl struct {
	store store.Store

	rulesByName map[string]map[string]*storage.ComplianceOperatorRule
	ruleLock    sync.RWMutex
}

func (d *datastoreImpl) Walk(ctx context.Context, fn func(rule *storage.ComplianceOperatorRule) error) error {
	if ok, err := complianceOperatorSAC.ReadAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.Wrap(sac.ErrResourceAccessDenied, "compliance operator rules read")
	}
	// Postgres retry in caller.
	return d.store.Walk(ctx, fn)
}

func (d *datastoreImpl) addToRulesByNameNoLock(rule *storage.ComplianceOperatorRule) {
	m := d.rulesByName[rule.GetName()]
	if m == nil {
		m = make(map[string]*storage.ComplianceOperatorRule)
		d.rulesByName[rule.GetName()] = m
	}
	m[rule.GetId()] = rule
}

func (d *datastoreImpl) Upsert(ctx context.Context, rule *storage.ComplianceOperatorRule) error {
	if ok, err := complianceOperatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.Wrap(sac.ErrResourceAccessDenied, "compliance operator rules write")
	}
	d.ruleLock.Lock()
	defer d.ruleLock.Unlock()

	if err := d.store.Upsert(ctx, rule); err != nil {
		return err
	}
	d.addToRulesByNameNoLock(rule)
	return nil
}

func (d *datastoreImpl) Delete(ctx context.Context, id string) error {
	if ok, err := complianceOperatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.Wrap(sac.ErrResourceAccessDenied, "compliance operator rules write")
	}

	d.ruleLock.Lock()
	defer d.ruleLock.Unlock()

	rule, exists, err := d.store.Get(ctx, id)
	if err != nil || !exists {
		return err
	}

	if err := d.store.Delete(ctx, rule.GetId()); err != nil {
		return err
	}
	delete(d.rulesByName[rule.GetName()], rule.GetId())
	return nil
}

func (d *datastoreImpl) GetRulesByName(ctx context.Context, name string) ([]*storage.ComplianceOperatorRule, error) {
	if ok, err := complianceOperatorSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.Wrap(sac.ErrResourceAccessDenied, "compliance operator rules read")
	}
	d.ruleLock.RLock()
	defer d.ruleLock.RUnlock()
	rules := make([]*storage.ComplianceOperatorRule, 0, len(d.rulesByName[name]))
	for _, rule := range d.rulesByName[name] {
		rules = append(rules, rule.Clone())
	}
	return rules, nil
}

func (d *datastoreImpl) ExistsByName(ctx context.Context, name string) (bool, error) {
	if ok, err := complianceOperatorSAC.ReadAllowed(ctx); err != nil {
		return false, err
	} else if !ok {
		return false, errors.Wrap(sac.ErrResourceAccessDenied, "compliance operator rules read")
	}
	d.ruleLock.RLock()
	defer d.ruleLock.RUnlock()
	val, ok := d.rulesByName[name]
	return ok && len(val) != 0, nil
}
