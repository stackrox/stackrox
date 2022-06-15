package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/config/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
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
	if privateConf := config.GetPrivateConfig(); privateConf != nil {
		if clusterRetentionConf := privateConf.GetDecommissionedClusterRetention(); clusterRetentionConf != nil {
			hasUpdate, err := d.hasClusterRetentionConfigUpdate(ctx, clusterRetentionConf)
			if err != nil {
				return err
			}

			if hasUpdate {
				clusterRetentionConf.LastUpdated = types.TimestampNow()
			}
		}
	}
	return d.store.Upsert(ctx, config)
}

func (d *datastoreImpl) hasClusterRetentionConfigUpdate(ctx context.Context,
	newConf *storage.DecommissionedClusterRetentionConfig) (bool, error) {
	conf, err := d.getClusterRetentionConfig(ctx)
	if err != nil {
		return false, err
	}
	return !clusterRetentionConfigsEqual(conf, newConf), nil
}

func (d *datastoreImpl) getClusterRetentionConfig(ctx context.Context) (*storage.DecommissionedClusterRetentionConfig, error) {
	conf, err := d.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	if privateConf := conf.GetPrivateConfig(); privateConf != nil {
		return privateConf.GetDecommissionedClusterRetention(), nil
	}
	return nil, nil
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
		ignoreLabelsEqual(c1.GetIgnoreLabel(), c2.GetIgnoreLabel())
}

func ignoreLabelsEqual(l1 *storage.DecommissionedClusterRetentionConfig_IgnoreClusterLabel,
	l2 *storage.DecommissionedClusterRetentionConfig_IgnoreClusterLabel) bool {
	if l1 == nil && l2 == nil {
		return true
	}
	if l1 == nil || l2 == nil {
		return false
	}
	return l1.GetKey() == l2.GetKey() && l1.GetValue() == l2.GetValue()
}
