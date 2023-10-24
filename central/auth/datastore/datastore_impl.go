package datastore

import (
	"context"

	"github.com/stackrox/rox/central/auth/m2m"
	"github.com/stackrox/rox/central/auth/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	_ DataStore = (*datastoreImpl)(nil)

	accessSAC = sac.ForResource(resources.Access)
)

type datastoreImpl struct {
	store store.Store
	set   m2m.TokenExchangerSet

	mutex sync.RWMutex
}

func (d *datastoreImpl) GetAuthM2MConfig(ctx context.Context, id string) (*storage.AuthMachineToMachineConfig, bool, error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.getAuthM2MConfigNoLock(ctx, id)
}

func (d *datastoreImpl) getAuthM2MConfigNoLock(ctx context.Context, id string) (*storage.AuthMachineToMachineConfig, bool, error) {
	return d.store.Get(ctx, id)
}

func (d *datastoreImpl) ListAuthM2MConfigs(ctx context.Context) ([]*storage.AuthMachineToMachineConfig, error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.store.GetAll(ctx)
}

func (d *datastoreImpl) AddAuthM2MConfig(ctx context.Context, config *storage.AuthMachineToMachineConfig) (*storage.AuthMachineToMachineConfig, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	exchanger, err := d.set.NewTokenExchangerFromConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	if err := d.store.Upsert(ctx, config); err != nil {
		return nil, err
	}

	if err := d.set.UpsertTokenExchanger(exchanger, config.GetIssuer()); err != nil {
		return nil, err
	}
	return config, nil
}

func (d *datastoreImpl) UpdateAuthM2MConfig(ctx context.Context, config *storage.AuthMachineToMachineConfig) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	existingConfig, exists, err := d.getAuthM2MConfigNoLock(ctx, config.GetId())
	if err != nil {
		return err
	}

	exchanger, err := d.set.NewTokenExchangerFromConfig(ctx, config)
	if err != nil {
		return err
	}

	if err := d.store.Upsert(ctx, config); err != nil {
		return err
	}

	if err := d.set.UpsertTokenExchanger(exchanger, config.GetIssuer()); err != nil {
		return err
	}

	// We need to ensure that any previously existing config is removed from the token exchanger set.
	// Since this updated config may have updated the issuer, we need to fetch the existing, stored config from the
	// database and ensure it's removed properly from the set. We do this at the end since we want the new config
	// to successfully exist beforehand.
	if exists && config.GetIssuer() != existingConfig.GetIssuer() {
		if err := d.set.RemoveTokenExchanger(existingConfig.GetIssuer()); err != nil {
			return err
		}
	}

	return nil
}

func (d *datastoreImpl) GetTokenExchanger(ctx context.Context, issuer string) (m2m.TokenExchanger, bool) {
	if err := sac.VerifyAuthzOK(accessSAC.ReadAllowed(ctx)); err != nil {
		return nil, false
	}
	return d.set.GetTokenExchanger(issuer)
}

func (d *datastoreImpl) RemoveAuthM2MConfig(ctx context.Context, id string) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	config, exists, err := d.getAuthM2MConfigNoLock(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if err := d.set.RemoveTokenExchanger(config.GetIssuer()); err != nil {
		return err
	}

	return d.store.Delete(ctx, id)
}
