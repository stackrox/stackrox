package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/networkgraph/config/datastore/internal/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	networkGraphConfigKey = "networkGraphConfig"
)

var (
	graphConfigSAC = sac.ForResource(resources.NetworkGraphConfig)
	log            = logging.LoggerForModule()
)

type datastoreImpl struct {
	store store.Store
}

// New return new instance of DataStore.
func New(s store.Store) DataStore {
	ds := &datastoreImpl{
		store: s,
	}

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraphConfig),
		))

	if err := ds.initDefaultConfig(ctx); err != nil {
		utils.Should(errors.Wrap(err, "could not initialize default network graph configuration"))
	}

	return ds
}

func (d *datastoreImpl) initDefaultConfig(ctx context.Context) error {
	_, found, err := d.store.Get(ctx, networkGraphConfigKey)
	if err != nil {
		return err
	}

	if !found {
		defaultConfig := &storage.NetworkGraphConfig{
			Id:                      networkGraphConfigKey,
			HideDefaultExternalSrcs: false,
		}
		if err := d.store.Upsert(ctx, defaultConfig); err != nil {
			return err
		}
	}
	return nil
}

func (d *datastoreImpl) GetNetworkGraphConfig(ctx context.Context) (*storage.NetworkGraphConfig, error) {
	if ok, err := graphConfigSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, sac.ErrResourceAccessDenied
	}

	config, found, err := d.store.Get(ctx, networkGraphConfigKey)
	if err != nil {
		return nil, err
	} else if !found {
		return nil, errors.New("graph configuration not found")
	}

	return config, nil
}

func (d *datastoreImpl) UpdateNetworkGraphConfig(ctx context.Context, config *storage.NetworkGraphConfig) error {
	if ok, err := graphConfigSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	config.Id = networkGraphConfigKey
	return d.store.Upsert(ctx, config)
}
