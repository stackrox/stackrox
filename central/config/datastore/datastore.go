package datastore

import (
	"context"
	"reflect"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/config/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
)

// DataStore is the entry point for modifying Config data.
//go:generate mockgen-wrapper
type DataStore interface {
	GetConfig(context.Context) (*storage.Config, error)
	UpsertConfig(context.Context, *storage.Config) error
}

// New returns an instance of DataStore.
func New(store store.Store) DataStore {
	return &datastoreImpl{
		store: store,
	}
}

var (
	configSAC = sac.ForResource(resources.Config)
)

type datastoreImpl struct {
	store store.Store
}

// GetConfig returns Central's config
func (d *datastoreImpl) GetConfig(ctx context.Context) (*storage.Config, error) {
	if ok, err := configSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	conf, _, err := d.store.Get(ctx)
	return conf, err
}

// UpsertConfig updates Central's config
func (d *datastoreImpl) UpsertConfig(ctx context.Context, config *storage.Config) error {
	if ok, err := configSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if features.DecommissionedClusterRetention.Enabled() {
		if clusterRetentionConf := config.GetPrivateConfig().GetDecommissionedClusterRetention(); clusterRetentionConf != nil {
			oldConf, err := d.getClusterRetentionConfig(ctx)
			if err != nil {
				return err
			}
			if oldConf != nil {
				clusterRetentionConf.CreatedAt = oldConf.GetCreatedAt()
			} else {
				clusterRetentionConf.CreatedAt = types.TimestampNow()
			}

			hasUpdate := !clusterRetentionConfigsEqual(oldConf, clusterRetentionConf)

			if hasUpdate {
				clusterRetentionConf.LastUpdated = types.TimestampNow()
			}
		}
	} else {
		if config.GetPrivateConfig() != nil {
			config.GetPrivateConfig().DecommissionedClusterRetention = nil
		}
	}

	return d.store.Upsert(ctx, config)
}

func (d *datastoreImpl) getClusterRetentionConfig(ctx context.Context) (*storage.DecommissionedClusterRetentionConfig, error) {
	conf, err := d.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	return conf.GetPrivateConfig().GetDecommissionedClusterRetention(), nil
}

func clusterRetentionConfigsEqual(c1 *storage.DecommissionedClusterRetentionConfig,
	c2 *storage.DecommissionedClusterRetentionConfig) bool {
	if c1 == nil && c2 == nil {
		return true
	}
	if c1 == nil || c2 == nil {
		return false
	}
	return c1.GetRetentionDurationDays() == c2.GetRetentionDurationDays() &&
		reflect.DeepEqual(c1.GetIgnoreClusterLabels(), c2.GetIgnoreClusterLabels())
}
