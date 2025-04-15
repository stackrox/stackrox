package datastore

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/stackrox/rox/central/config/store"
	pgStore "github.com/stackrox/rox/central/config/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
	"k8s.io/apimachinery/pkg/util/sets"
)

var (
	log = logging.LoggerForModule()
)

// DataStore is the entry point for modifying Config data.
//
//go:generate mockgen-wrapper
type DataStore interface {
	GetConfig(context.Context) (*storage.Config, error)
	GetPrivateConfig(context.Context) (*storage.PrivateConfig, error)
	GetVulnerabilityExceptionConfig(ctx context.Context) (*storage.VulnerabilityExceptionConfig, error)
	GetPublicConfig() (*storage.PublicConfig, error)
	UpsertConfig(context.Context, *storage.Config) error

	GetPlatformComponentConfig(context.Context) (*storage.PlatformComponentConfig, bool, error)
	UpsertPlatformComponentConfigRule(context.Context, *storage.PlatformComponentConfig_Rule) error
	UpsertPlatformComponentConfigRules(context.Context, []*storage.PlatformComponentConfig_Rule) (*storage.PlatformComponentConfig, error)
	DeletePlatformComponentConfigRules(context.Context, ...string) error
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
	vmRequestsSAC     = sac.ForResource(resources.VulnerabilityManagementRequests)
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

func (d *datastoreImpl) GetVulnerabilityExceptionConfig(ctx context.Context) (*storage.VulnerabilityExceptionConfig, error) {
	if ok, err := administrationSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		if ok, err := vmRequestsSAC.ReadAllowed(ctx); err != nil {
			return nil, err
		} else if !ok {
			return nil, nil
		}
	}

	adminCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))
	conf, _, err := d.store.Get(adminCtx)
	return conf.GetPrivateConfig().GetVulnerabilityExceptionConfig(), err
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
			clusterRetentionConf.CreatedAt = protocompat.TimestampNow()
		}

		hasUpdate := !clusterRetentionConfigsEqual(oldConf, clusterRetentionConf)

		if hasUpdate {
			clusterRetentionConf.LastUpdated = protocompat.TimestampNow()
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

func (d *datastoreImpl) GetPlatformComponentConfig(ctx context.Context) (*storage.PlatformComponentConfig, bool, error) {
	if ok, err := administrationSAC.ReadAllowed(ctx); err != nil {
		return nil, false, err
	} else if !ok {
		return nil, false, nil
	}

	adminCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))

	config, found, err := d.store.Get(adminCtx)
	if config == nil {
		return nil, false, nil
	}
	return config.GetPlatformComponentConfig(), found && config.GetPlatformComponentConfig() != nil, err
}

func (d *datastoreImpl) UpsertPlatformComponentConfigRule(ctx context.Context, rule *storage.PlatformComponentConfig_Rule) error {
	if ok, err := administrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return nil
	}

	adminCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))

	config, found, err := d.store.Get(adminCtx)
	if !found || err != nil {
		return err
	}
	if config.PlatformComponentConfig.Rules == nil {
		config.PlatformComponentConfig.Rules = make([]*storage.PlatformComponentConfig_Rule, 0)
	}
	config.PlatformComponentConfig.Rules = append(config.PlatformComponentConfig.Rules, rule)
	return d.store.Upsert(adminCtx, config)
}

func (d *datastoreImpl) UpsertPlatformComponentConfigRules(ctx context.Context, rules []*storage.PlatformComponentConfig_Rule) (*storage.PlatformComponentConfig, error) {
	if ok, err := administrationSAC.WriteAllowed(ctx); err != nil {
		log.Info("Error while checking permission")
		return nil, err
	} else if !ok {
		log.Info("User did not have write access to the administration resource")
		return nil, nil
	}

	adminCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))

	config, found, err := d.store.Get(adminCtx)
	if !found || err != nil {
		log.Info("Config not found or there was an error")
		return nil, err
	}
	if config.PlatformComponentConfig.Rules == nil {
		log.Info("config.PlatformComponentConfig.Rules was empty")
		config.PlatformComponentConfig.Rules = make([]*storage.PlatformComponentConfig_Rule, 0)
	}
	ruleNameSet := sets.NewString()
	for _, rule := range config.PlatformComponentConfig.Rules {
		ruleNameSet.Insert(rule.Name)
	}
	for _, rule := range rules {
		if ruleNameSet.Has(rule.Name) {
			continue
		}
		log.Infof("Rule: %q", rule)
		config.PlatformComponentConfig.Rules = append(config.PlatformComponentConfig.Rules, rule)
	}
	err = d.store.Upsert(adminCtx, config)
	if err != nil {
		log.Info("There was an error upserting the config")
		return nil, err
	}
	log.Infof("Config after upsert: %q", config)
	return config.PlatformComponentConfig, nil
}

func (d *datastoreImpl) DeletePlatformComponentConfigRules(ctx context.Context, rules ...string) error {
	if ok, err := administrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return nil
	}

	adminCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))

	config, found, err := d.store.Get(adminCtx)
	if !found || err != nil {
		return err
	}
	if config.PlatformComponentConfig.Rules == nil {
		config.PlatformComponentConfig.Rules = make([]*storage.PlatformComponentConfig_Rule, 0)
	}

	ruleNameSet := sets.NewString(rules...)
	newRules := make([]*storage.PlatformComponentConfig_Rule, 0)
	for _, rule := range config.PlatformComponentConfig.Rules {
		if ruleNameSet.Has(rule.GetName()) {
			continue
		}
		newRules = append(newRules, rule)
	}
	config.PlatformComponentConfig.Rules = newRules
	return d.store.Upsert(adminCtx, config)
}
