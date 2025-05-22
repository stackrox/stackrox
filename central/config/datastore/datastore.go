package datastore

import (
	"context"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/config/store"
	pgStore "github.com/stackrox/rox/central/config/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
)

// DataStore is the entry point for modifying Config data.
//
//go:generate mockgen-wrapper
type DataStore interface {
	GetConfig(ctx context.Context) (*storage.Config, error)
	GetPrivateConfig(ctx context.Context) (*storage.PrivateConfig, error)
	GetVulnerabilityExceptionConfig(ctx context.Context) (*storage.VulnerabilityExceptionConfig, error)
	GetPublicConfig() (*storage.PublicConfig, error)
	UpsertConfig(ctx context.Context, config *storage.Config) error

	GetPlatformComponentConfig(ctx context.Context) (*storage.PlatformComponentConfig, bool, error)
	GetDefaultRedHatLayeredProductsRegex() string
	UpsertPlatformComponentConfigRules(ctx context.Context, rules []*storage.PlatformComponentConfig_Rule) (*storage.PlatformComponentConfig, error)
	MarkPCCReevaluated(context.Context) error
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
	if config.GetPlatformComponentConfig() != nil {
		existingPlatformConf, _, _ := d.GetPlatformComponentConfig(ctx)
		platformConfig, err := validateAndUpdatePlatformComponentConfig(existingPlatformConf, config.GetPlatformComponentConfig().GetRules())
		if err != nil {
			return err
		}
		config.PlatformComponentConfig = platformConfig
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

	config, found, err := d.store.Get(ctx)
	if config == nil {
		return nil, false, nil
	}
	return config.GetPlatformComponentConfig(), found && config.GetPlatformComponentConfig() != nil, err
}

func (d *datastoreImpl) UpsertPlatformComponentConfigRules(ctx context.Context, rules []*storage.PlatformComponentConfig_Rule) (*storage.PlatformComponentConfig, error) {
	if ok, err := administrationSAC.WriteAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, sac.ErrResourceAccessDenied
	}

	config, found, err := d.store.Get(ctx)
	if !found {
		return nil, errors.Wrap(errox.NotFound, "System configuration not found")
	} else if err != nil {
		return nil, err
	}

	config.PlatformComponentConfig, err = validateAndUpdatePlatformComponentConfig(config.PlatformComponentConfig, rules)
	if err != nil {
		return nil, err
	}
	err = d.store.Upsert(ctx, config)
	if err != nil {
		return nil, err
	}
	return config.GetPlatformComponentConfig(), nil
}

func (_ *datastoreImpl) GetDefaultRedHatLayeredProductsRegex() string {
	return defaultPlatformConfigLayeredProductsRule.NamespaceRule.Regex
}

func validateAndUpdatePlatformComponentConfig(config *storage.PlatformComponentConfig, rules []*storage.PlatformComponentConfig_Rule) (*storage.PlatformComponentConfig, error) {
	systemRuleExists := false
	layeredProductsRuleExists := false
	parsedRules := make([]*storage.PlatformComponentConfig_Rule, 0)
	for _, rule := range rules {
		if rule.Name == defaultPlatformConfigSystemRule.Name && !strings.EqualFold(rule.NamespaceRule.Regex, defaultPlatformConfigSystemRule.NamespaceRule.Regex) {
			// Prevent override of system rule
			return nil, errors.New("System rule cannot be overwritten")
		} else if rule.Name == defaultPlatformConfigSystemRule.Name && strings.EqualFold(rule.NamespaceRule.Regex, defaultPlatformConfigSystemRule.NamespaceRule.Regex) {
			// If for some reason they're trying to duplicate the system rule, we prevent that
			if systemRuleExists {
				continue
			}
			systemRuleExists = true
		}
		if rule.Name == defaultPlatformConfigLayeredProductsRule.Name {
			// If for some reason somebody makes a rule with an identical name to the layered products rule, we only take the first occurrence of it.
			if layeredProductsRuleExists {
				continue
			}
			layeredProductsRuleExists = true
		}
		parsedRules = append(parsedRules, rule)
	}
	// Add back in default rules if they weren't passed in by the user
	if !systemRuleExists {
		parsedRules = append(parsedRules, defaultPlatformConfigSystemRule)
	}
	if !layeredProductsRuleExists {
		parsedRules = append(parsedRules, defaultPlatformConfigLayeredProductsRule)
	}
	if config == nil {
		config = &storage.PlatformComponentConfig{
			Rules:             parsedRules,
			NeedsReevaluation: true,
		}
	} else {
		slices.SortFunc(config.GetRules(), ruleNameSortFunc)
		slices.SortFunc(parsedRules, ruleNameSortFunc)
		if !protoutils.SlicesEqual(config.GetRules(), parsedRules) {
			config.NeedsReevaluation = true
		}
		config.Rules = parsedRules
	}
	return config, nil
}

func ruleNameSortFunc(a *storage.PlatformComponentConfig_Rule, b *storage.PlatformComponentConfig_Rule) int {
	return strings.Compare(a.Name, b.Name)
}

func (d *datastoreImpl) MarkPCCReevaluated(ctx context.Context) error {
	if ok, err := administrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return nil
	}

	config, found, err := d.store.Get(ctx)
	if !found || err != nil {
		return err
	}

	config.GetPlatformComponentConfig().NeedsReevaluation = false
	return d.store.Upsert(ctx, config)
}
