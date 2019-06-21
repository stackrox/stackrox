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
	err = ds.setEnabledAuthzPluginUnlocked(enabledConfig)
	if err != nil {
		log.Errorf("Authorization plugin is not configured properly on initialization: %v.  API "+
			"requests will be rejected until authorization plugin configuration is fixed.  Please log in with "+
			"username/password to fix the configuration", err)
		ds.setErrorAuthzPluginUnlocked(enabledConfig, errors.New("authorization plugin is not configured properly"))
	}
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

	// Determine insert or update
	if config.GetId() == "" {
		config.Id = uuid.NewV4().String()
	} else {
		if existingConfig, err := ds.storage.GetAuthzPluginConfig(config.GetId()); err != nil {
			return nil, err
		} else if existingConfig == nil {
			return nil, errors.Errorf("cannot update non-existent auth plugin config with id %s", config.GetId())
		}
	}

	oldEnabledPlugin := ds.enabledPlugin
	// Validate the plugin config and set the current auth plugin client
	if !config.GetEnabled() && oldEnabledPlugin != nil && oldEnabledPlugin.GetId() == config.GetId() {
		// We are turning off the previously enabled plugin
		err := ds.setEnabledAuthzPluginUnlocked(nil)
		if err != nil {
			return nil, err
		}
	} else if config.GetEnabled() {
		err := ds.setEnabledAuthzPluginUnlocked(config)
		if err != nil {
			return nil, err
		}
	}

	// Store the new plugin config
	err := ds.storage.UpsertAuthzPluginConfig(config)
	if err != nil {
		return nil, err
	}

	// Disable the previously enabled config if necessary
	if config.GetEnabled() && oldEnabledPlugin != nil && oldEnabledPlugin.GetId() != config.GetId() {
		oldEnabledPlugin.Enabled = false
		if err := ds.storage.UpsertAuthzPluginConfig(oldEnabledPlugin); err != nil {
			return nil, err
		}
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
		err := ds.setEnabledAuthzPluginUnlocked(nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ds *datastoreImpl) setEnabledAuthzPluginUnlocked(config *storage.AuthzPluginConfig) error {
	if config == nil {
		ds.clientMgr.SetClient(nil)
		ds.enabledPlugin = nil
		return nil
	}

	newClient, err := client.New(config.GetEndpointConfig())
	if err != nil {
		return err
	}
	ds.clientMgr.SetClient(newClient)
	ds.enabledPlugin = config
	return nil
}

// Use an auto-fail client but still track which config is enabled
func (ds *datastoreImpl) setErrorAuthzPluginUnlocked(config *storage.AuthzPluginConfig, err error) {
	ds.clientMgr.SetClient(client.NewErrorClient(err))
	ds.enabledPlugin = config
}
