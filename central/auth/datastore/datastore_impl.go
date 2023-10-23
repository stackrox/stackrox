package datastore

import (
	"context"

	"github.com/stackrox/rox/central/auth/store"
	"github.com/stackrox/rox/generated/storage"
)

var (
	_ DataStore = (*datastoreImpl)(nil)
)

type datastoreImpl struct {
	store store.Store
}

func (d *datastoreImpl) GetAuthM2MConfig(ctx context.Context, id string) (*storage.AuthMachineToMachineConfig, bool, error) {
	return d.store.Get(ctx, id)
}

func (d *datastoreImpl) ListAuthM2MConfigs(ctx context.Context) ([]*storage.AuthMachineToMachineConfig, error) {
	return d.store.GetAll(ctx)
}

func (d *datastoreImpl) AddAuthM2MConfig(ctx context.Context, config *storage.AuthMachineToMachineConfig) (*storage.AuthMachineToMachineConfig, error) {
	setIssuer(config)
	if err := d.store.Upsert(ctx, config); err != nil {
		return nil, err
	}
	return config, nil
}

func (d *datastoreImpl) UpdateAuthM2MConfig(ctx context.Context, config *storage.AuthMachineToMachineConfig) error {
	setIssuer(config)
	if err := d.store.Upsert(ctx, config); err != nil {
		return err
	}
	return nil
}

func (d *datastoreImpl) RemoveAuthM2MConfig(ctx context.Context, id string) error {
	return d.store.Delete(ctx, id)
}

func setIssuer(config *storage.AuthMachineToMachineConfig) {
	switch config.GetType() {
	case storage.AuthMachineToMachineConfig_GITHUB_ACTIONS:
		// This allows to set a custom issuer in case e.g. GitHub cloud or enterprise are used.
		// Ref: https://docs.github.com/en/enterprise-cloud@latest/rest/actions/oidc?apiVersion=2022-11-28
		if config.GetIssuer() == "" {
			config.Issuer = "https://token.actions.githubusercontent.com"
		}
	}
}
