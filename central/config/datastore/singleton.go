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

	if needsUpsert {
		utils.Must(datastore.UpsertConfig(ctx, &storage.Config{
			PublicConfig:  config.GetPublicConfig(),
			PrivateConfig: privateConfig,
		}))
	}
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
