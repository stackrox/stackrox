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
	networkGraphSAC = sac.ForResource(resources.NetworkGraph)
	log             = logging.LoggerForModule()
)

type datastoreImpl struct {
	store store.Store
}

// New return new instance of DataStore.
func New(storage store.Store) DataStore {
	ds := &datastoreImpl{
		store: storage,
	}

	if err := ds.initDefaultConfig(); err != nil {
		utils.Should(errors.Wrap(err, "could not initialize default network graph configuration"))
	}

	return ds
}

func (d *datastoreImpl) initDefaultConfig() error {
	_, found, err := d.store.Get(networkGraphConfigKey)
	if err != nil {
		return err
	}

	if !found {
		defaultConfig := &storage.NetworkGraphConfig{
			HideDefaultExternalSrcs: false,
		}
		if err := d.store.UpsertWithID(networkGraphConfigKey, defaultConfig); err != nil {
			return err
		}
	}
	return nil
}

func (d *datastoreImpl) GetNetworkGraphConfig(ctx context.Context) (*storage.NetworkGraphConfig, error) {
	if ok, err := networkGraphSAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, err
	}

	config, found, err := d.store.Get(networkGraphConfigKey)
	if err != nil {
		return nil, err
	}

	if !found {
		return &storage.NetworkGraphConfig{
			HideDefaultExternalSrcs: false,
		}, nil
	}
	return config, nil
}

func (d *datastoreImpl) UpdateNetworkGraphConfig(ctx context.Context, config *storage.NetworkGraphConfig) error {
	if ok, err := networkGraphSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return d.store.UpsertWithID(networkGraphConfigKey, config)
}
