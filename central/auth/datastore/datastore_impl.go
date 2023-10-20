package datastore

import (
	"context"

	"github.com/stackrox/rox/central/auth/m2m"
	"github.com/stackrox/rox/central/auth/store"
	"github.com/stackrox/rox/generated/storage"
)

var (
	_ DataStore = (*datastoreImpl)(nil)
)

type datastoreImpl struct {
	store store.Store
	set   m2m.TokenExchangerSet
}

func (d *datastoreImpl) GetAuthM2MConfig(ctx context.Context, id string) (*storage.AuthMachineToMachineConfig, bool, error) {
	return d.store.Get(ctx, id)
}

func (d *datastoreImpl) ListAuthM2MConfigs(ctx context.Context) ([]*storage.AuthMachineToMachineConfig, error) {
	return d.store.GetAll(ctx)
}

func (d *datastoreImpl) AddAuthM2MConfig(ctx context.Context, config *storage.AuthMachineToMachineConfig) (*storage.AuthMachineToMachineConfig, error) {
	if err := d.store.Upsert(ctx, config); err != nil {
		return nil, err
	}

	if err := d.set.UpsertTokenExchanger(ctx, config); err != nil {
		return nil, err
	}
	return config, nil
}

func (d *datastoreImpl) UpdateAuthM2MConfig(ctx context.Context, config *storage.AuthMachineToMachineConfig) error {
	if err := d.store.Upsert(ctx, config); err != nil {
		return err
	}

	if err := d.set.UpsertTokenExchanger(ctx, config); err != nil {
		return err
	}
	return nil
}

func (d *datastoreImpl) RemoveAuthM2MConfig(ctx context.Context, id string) error {
	if err := d.store.Delete(ctx, id); err != nil {
		return err
	}

	return d.set.RemoveTokenExchanger(id)
}
