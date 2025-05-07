package datastore

import (
	"context"

	pgStore "github.com/stackrox/rox/central/config/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	// DefaultDeployAlertRetention is the number of days to retain resolved deployment alerts.
	DefaultDeployAlertRetention = 7
	// DefaultRuntimeAlertRetention is the number of days to retain all runtime alerts.
	DefaultRuntimeAlertRetention = 30
	// DefaultDeletedRuntimeAlertRetention is the number of days to retain runtime alerts for deleted deployments.
	DefaultDeletedRuntimeAlertRetention = 7
	// DefaultImageRetention is the number of days to retain images for.
	DefaultImageRetention = 7
	// DefaultAttemptedDeployAlertRetention is the number of days to retain all attempted deploy-time alerts.
	DefaultAttemptedDeployAlertRetention = 7
	// DefaultAttemptedRuntimeAlertRetention is the number of days to retain all attempted run-time alerts.
	DefaultAttemptedRuntimeAlertRetention = 7
	// DefaultExpiredVulnReqRetention is the number of days to retain expired vulnerability requests.
	DefaultExpiredVulnReqRetention = 90
	// DefaultDecommissionedClusterRetentionDays is the number of days to retain a cluster that is unreachable.
	DefaultDecommissionedClusterRetentionDays = 0
	// DefaultReportHistoryRetentionWindow number of days to retain reports.
	DefaultReportHistoryRetentionWindow = 7
	// DefaultDownloadableReportRetentionDays number of days to retain downloadable reports.
	DefaultDownloadableReportRetentionDays = 7
	// DefaultDownloadableReportGlobalRetentionBytes is the maximum total upper limit in bytes for all downloadable reports.
	DefaultDownloadableReportGlobalRetentionBytes = 500 * 1024 * 1024
	// DefaultAdministrationEventsRetention is the number of days to retain administration events.
	DefaultAdministrationEventsRetention = 4
	// PlatformComponentSystemRuleName is the name of the system defined rule for matching openshift and kube workloads
	PlatformComponentSystemRuleName = "system rule"
	// PlatformComponentSystemRegex is the system defined regex for matching kube and openshift workloads
	PlatformComponentSystemRegex = `^kube-.*|^openshift-.*`
	// PlatformComponentLayeredProductsRuleName is the name of the system defined rule for matching workloads created by Red hat layered products
	PlatformComponentLayeredProductsRuleName = "red hat layered products"
	// PlatformComponentLayeredProductsDefaultRegex is the default regex for matching workloads created by Red hat layered products
	PlatformComponentLayeredProductsDefaultRegex = `^stackrox$|^rhacs-operator$|^open-cluster-management$|^multicluster-engine$|^aap$|^hive$`
)

var (
	once sync.Once

	d DataStore

	defaultAlertRetention = &storage.PrivateConfig_AlertConfig{
		AlertConfig: &storage.AlertRetentionConfig{
			ResolvedDeployRetentionDurationDays:   DefaultDeployAlertRetention,
			DeletedRuntimeRetentionDurationDays:   DefaultDeletedRuntimeAlertRetention,
			AllRuntimeRetentionDurationDays:       DefaultRuntimeAlertRetention,
			AttemptedDeployRetentionDurationDays:  DefaultAttemptedDeployAlertRetention,
			AttemptedRuntimeRetentionDurationDays: DefaultAttemptedRuntimeAlertRetention,
		},
	}

	defaultPrivateConfig = storage.PrivateConfig{
		ImageRetentionDurationDays:          DefaultImageRetention,
		AlertRetention:                      defaultAlertRetention,
		ExpiredVulnReqRetentionDurationDays: DefaultExpiredVulnReqRetention,
	}

	defaultVulnerabilityDeferralConfig = &storage.VulnerabilityExceptionConfig{
		ExpiryOptions: &storage.VulnerabilityExceptionConfig_ExpiryOptions{
			DayOptions: []*storage.DayOption{
				{
					NumDays: 14,
					Enabled: true,
				},
				{
					NumDays: 30,
					Enabled: true,
				},
				{
					NumDays: 60,
					Enabled: true,
				},
				{
					NumDays: 90,
					Enabled: true,
				},
			},
			FixableCveOptions: &storage.VulnerabilityExceptionConfig_FixableCVEOptions{
				AllFixable: true,
				AnyFixable: true,
			},
			CustomDate: false,
			Indefinite: false,
		},
	}

	defaultDecommissionedClusterRetention = &storage.DecommissionedClusterRetentionConfig{
		RetentionDurationDays: DefaultDecommissionedClusterRetentionDays,
	}

	defaultReportRetentionConfig = &storage.ReportRetentionConfig{
		HistoryRetentionDurationDays:           DefaultReportHistoryRetentionWindow,
		DownloadableReportRetentionDays:        DefaultDownloadableReportRetentionDays,
		DownloadableReportGlobalRetentionBytes: DefaultDownloadableReportGlobalRetentionBytes,
	}

	defaultAdministrationEventsConfig = &storage.AdministrationEventsConfig{
		RetentionDurationDays: DefaultAdministrationEventsRetention,
	}

	defaultPlatformConfigSystemRule = &storage.PlatformComponentConfig_Rule{
		Name: PlatformComponentSystemRuleName,
		NamespaceRule: &storage.PlatformComponentConfig_Rule_NamespaceRule{
			Regex: PlatformComponentSystemRegex,
		},
	}
	defaultPlatformConfigLayeredProductsRule = &storage.PlatformComponentConfig_Rule{
		Name: PlatformComponentLayeredProductsRuleName,
		NamespaceRule: &storage.PlatformComponentConfig_Rule_NamespaceRule{
			Regex: PlatformComponentLayeredProductsDefaultRegex,
		},
	}
)

func validateConfigAndPopulateMissingDefaults(datastore DataStore) {
	ctx := sac.WithGlobalAccessScopeChecker(
		context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))
	config, err := datastore.GetConfig(ctx)
	if err != nil {
		panic(err)
	}
	if config == nil {
		config = &storage.Config{}
	}

	// See the note next to the publicConfigCache variable in datastore.go for
	// more information on public config caching.
	cachePublicConfig(config.GetPublicConfig())

	needsUpsert := false
	privateConfig := config.GetPrivateConfig()
	if privateConfig == nil {
		privateConfig = defaultPrivateConfig.CloneVT()
		needsUpsert = true
	}

	if privateConfig.GetDecommissionedClusterRetention() == nil {
		privateConfig.DecommissionedClusterRetention = defaultDecommissionedClusterRetention
		needsUpsert = true
	}

	if privateConfig.GetReportRetentionConfig() == nil {
		privateConfig.ReportRetentionConfig = defaultReportRetentionConfig
		needsUpsert = true
	}

	if features.UnifiedCVEDeferral.Enabled() {
		if privateConfig.GetVulnerabilityExceptionConfig() == nil {
			privateConfig.VulnerabilityExceptionConfig = defaultVulnerabilityDeferralConfig
			needsUpsert = true
		}
	}

	if privateConfig.GetAdministrationEventsConfig() == nil {
		privateConfig.AdministrationEventsConfig = defaultAdministrationEventsConfig
		needsUpsert = true
	}

	if features.CustomizablePlatformComponents.Enabled() && populateDefaultSystemRulesIfMissing(config) {
		needsUpsert = true
	}

	if needsUpsert {
		config.PrivateConfig = privateConfig
		utils.Must(datastore.UpsertConfig(ctx, config))
	}
}

// populateDefaultSystemRuleIfMissing returns true if the platform component config's system or layered products rules were updated
func populateDefaultSystemRulesIfMissing(config *storage.Config) bool {
	if config.GetPlatformComponentConfig() == nil {
		config.PlatformComponentConfig = &storage.PlatformComponentConfig{
			Rules: []*storage.PlatformComponentConfig_Rule{
				defaultPlatformConfigSystemRule,
				defaultPlatformConfigLayeredProductsRule},
		}
		return true
	}
	hasSystemRule := false
	hasLayeredProductsRule := false
	for _, rule := range config.GetPlatformComponentConfig().GetRules() {
		if rule.GetName() == PlatformComponentSystemRuleName {
			hasSystemRule = true
		} else if rule.GetName() == PlatformComponentLayeredProductsRuleName {
			hasLayeredProductsRule = true
		}
	}

	if hasSystemRule && hasLayeredProductsRule {
		return false
	}

	if !hasSystemRule {
		config.GetPlatformComponentConfig().Rules = append(
			config.GetPlatformComponentConfig().Rules,
			defaultPlatformConfigSystemRule,
		)
	}
	if !hasLayeredProductsRule {
		config.GetPlatformComponentConfig().Rules = append(
			config.GetPlatformComponentConfig().Rules,
			defaultPlatformConfigLayeredProductsRule,
		)
	}

	return true
}

func initialize() {
	store := pgStore.New(globaldb.GetPostgres())

	d = New(store)

	validateConfigAndPopulateMissingDefaults(d)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return d
}
