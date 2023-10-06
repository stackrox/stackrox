package datastore

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/stackrox/rox/central/config/store"
	pgStore "github.com/stackrox/rox/central/config/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
)

// DataStore is the entry point for modifying Config data.
//
//go:generate mockgen-wrapper
type DataStore interface {
	GetConfig(context.Context) (*storage.Config, error)
	GetPrivateConfig(context.Context) (*storage.PrivateConfig, error)
	GetPublicConfig() (*storage.PublicConfig, error)
	UpsertConfig(context.Context, *storage.Config) error
}

const (
	publicConfigCacheSize = 1
	publicConfigKey       = "public configuration"
)

var (
	// Notes on configuration caching:
	// - The public part of the configuration can be accessed from
	// an unauthenticated endpoint. In order to shield the database from
	// impacts of possible high traffic on that endpoint, the public
	// configuration data is served from cache.
	// - Currently, the cache is populated / refreshed at backend start,
	// on update, and on cache miss.
	// - In the future, considering multiple central instances will serve
	// traffic from the same database instance, the cache content is
	// expected to be eventually consistent. The instance receiving
	// config updates will be the first one to have a consistent cache,
	// other instances will be consistent after cached item expiration and
	// reload on cache miss.
	publicConfigCache *expirable.LRU[string, *storage.PublicConfig]

	cacheInitOnce sync.Once
)

func getPublicConfigCache() *expirable.LRU[string, *storage.PublicConfig] {
	cacheInitOnce.Do(func() {
		publicConfigCache = expirable.NewLRU[string, *storage.PublicConfig](
			publicConfigCacheSize,
			nil,
			1*time.Minute,
		)
	})
	return publicConfigCache
}

// New returns an instance of DataStore.
func New(store store.Store) DataStore {
	return &datastoreImpl{
		store: store,
	}
}

// NewForTest returns an instance of DataStore for testing purpose.
func NewForTest(_ *testing.T, db postgres.DB) DataStore {
	return New(pgStore.New(db))
}

var (
	administrationSAC = sac.ForResource(resources.Administration)
)

type datastoreImpl struct {
	store store.Store
}

// GetPublicConfig returns the public part of the Central config.
// The primary data source will be the cache and the secondary the database.
func (d *datastoreImpl) GetPublicConfig() (*storage.PublicConfig, error) {
	var err error
	// See the note next to the publicConfigCache variable for
	// more information on public config caching.
	publicConfig, found := getPublicConfigCache().Get(publicConfigKey)
	if found && publicConfig != nil {
		return publicConfig, nil
	}

	publicConfig, err = d.getPublicConfigFromDB()
	if err != nil {
		return publicConfig, err
	}

	// See the note next to the publicConfigCache variable for
	// more information on public config caching.
	cachePublicConfig(publicConfig)

	return publicConfig, err
}

func cachePublicConfig(publicConfig *storage.PublicConfig) {
	// The result of the cache addition is ignored as the information
	// whether the cache did evict anything is irrelevant.
	_ = getPublicConfigCache().Add(publicConfigKey, publicConfig)
}

// getPublicConfigFroDB returns the public part of the Central config
func (d *datastoreImpl) getPublicConfigFromDB() (*storage.PublicConfig, error) {
	elevatedCtx := sac.WithGlobalAccessScopeChecker(
		context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Administration),
		),
	)

	conf, _, err := d.store.Get(elevatedCtx)
	return conf.GetPublicConfig(), err
}

// GetPrivateConfig returns Central's config
func (d *datastoreImpl) GetPrivateConfig(ctx context.Context) (*storage.PrivateConfig, error) {
	if ok, err := administrationSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	conf, _, err := d.store.Get(ctx)
	return conf.GetPrivateConfig(), err
}

// GetConfig returns Central's config
func (d *datastoreImpl) GetConfig(ctx context.Context) (*storage.Config, error) {
	if ok, err := administrationSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	conf, _, err := d.store.Get(ctx)
	return conf, err
}

// UpsertConfig updates Central's config
func (d *datastoreImpl) UpsertConfig(ctx context.Context, config *storage.Config) error {
	if ok, err := administrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
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

	upsertErr := d.store.Upsert(ctx, config)
	if upsertErr != nil {
		return upsertErr
	}

	// See the note next to the publicConfigCache variable for
	// more information on public config caching.
	cachePublicConfig(config.GetPublicConfig())
	return nil
}

func (d *datastoreImpl) getClusterRetentionConfig(ctx context.Context) (*storage.DecommissionedClusterRetentionConfig, error) {
	privateConf, err := d.GetPrivateConfig(ctx)
	if err != nil {
		return nil, err
	}
	return privateConf.GetDecommissionedClusterRetention(), nil
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
