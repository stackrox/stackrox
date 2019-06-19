package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/role/resources"
	centralSAC "github.com/stackrox/rox/central/sac"
	"github.com/stackrox/rox/central/sac/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/client"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	authPluginSAC = sac.ForResource(resources.AuthPlugin)
)

type datastoreImpl struct {
	storage       store.Store
	clientMgr     centralSAC.AuthPluginClientManger
	enabledPlugin *storage.AuthzPluginConfig

	mutex sync.Mutex
}

func (ds *datastoreImpl) Initialize() error {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()
	// Enable auth client on startup
	plugins, err := ds.storage.ListAuthzPluginConfigs()
	if err != nil {
		return err
	}

	var enabledConfig *storage.AuthzPluginConfig
	for _, plugin := range plugins {
		if plugin.Enabled {
			if enabledConfig == nil {
				enabledConfig = plugin
				continue
			}
			log.Warnf("found multiple enabled auth plugins on init.  defaulting to %s:%s and disabling %s:%s",
				enabledConfig.GetName(),
				enabledConfig.GetId(),
				plugin.GetName(),
				plugin.GetId(),
			)
			plugin.Enabled = false
			err := ds.storage.UpsertAuthzPluginConfig(plugin)
			if err != nil {
				return err
			}
		}
	}
	ds.setEnabledAuthzPluginUnlocked(enabledConfig)
	return nil
}

func (ds *datastoreImpl) ListAuthzPluginConfigs(ctx context.Context) ([]*storage.AuthzPluginConfig, error) {
	if ok, err := authPluginSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.New("permission denied")
	}

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	return ds.storage.ListAuthzPluginConfigs()
}

func (ds *datastoreImpl) UpsertAuthzPluginConfig(ctx context.Context, config *storage.AuthzPluginConfig) (*storage.AuthzPluginConfig, error) {
	if ok, err := authPluginSAC.WriteAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.New("permission denied")
	}

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	if config.GetId() == "" {
		config.Id = uuid.NewV4().String()
	} else {
		if existingConfig, err := ds.storage.GetAuthzPluginConfig(config.GetId()); err != nil {
			return nil, err
		} else if existingConfig == nil {
			return nil, errors.Errorf("cannot update non-existent auth plugin config with id %s", config.GetId())
		}
	}
	if err := ds.storage.UpsertAuthzPluginConfig(config); err != nil {
		return nil, err
	}

	oldEnabledPlugin := ds.enabledPlugin
	// The upserted plugin is not enabled.  Figure out if we need to turn off a previously enabled plugin.
	if !config.GetEnabled() {
		if oldEnabledPlugin != nil && oldEnabledPlugin.GetId() == config.GetId() && oldEnabledPlugin.GetEnabled() {
			ds.setEnabledAuthzPluginUnlocked(nil)
		}
		return config, nil
	}

	// The upserted plugin is enabled.  Figure out if we need to turn off a previously enabled plugin.
	ds.setEnabledAuthzPluginUnlocked(config)
	if oldEnabledPlugin == nil || oldEnabledPlugin.GetId() == config.GetId() {
		return config, nil
	}
	oldEnabledPlugin.Enabled = false
	if err := ds.storage.UpsertAuthzPluginConfig(oldEnabledPlugin); err != nil {
		return nil, err
	}

	return config, nil
}

func (ds *datastoreImpl) DeleteAuthzPluginConfig(ctx context.Context, id string) error {
	if ok, err := authPluginSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	if err := ds.storage.DeleteAuthzPluginConfig(id); err != nil {
		return err
	}

	if ds.enabledPlugin.GetId() == id {
		ds.setEnabledAuthzPluginUnlocked(nil)
	}
	return nil
}

func (ds *datastoreImpl) setEnabledAuthzPluginUnlocked(config *storage.AuthzPluginConfig) {
	ds.enabledPlugin = config

	var newClient client.Client
	if config != nil {
		newClient = client.New(config)
	}
	ds.clientMgr.SetClient(newClient)
}
