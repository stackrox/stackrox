package datastore

import (
	"context"

	pgStore "github.com/stackrox/rox/central/config/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	// DefaultDeployAlertRetention is the number of days to retain resolved deployment alerts
	DefaultDeployAlertRetention = 7
	// DefaultRuntimeAlertRetention is the number of days to retain all runtime alerts
	DefaultRuntimeAlertRetention = 30
	// DefaultDeletedRuntimeAlertRetention is the number of days to retain runtime alerts for deleted deployments
	DefaultDeletedRuntimeAlertRetention = 7
	// DefaultImageRetention is the number of days to retain images for
	DefaultImageRetention = 7
	// DefaultAttemptedDeployAlertRetention is the number of days to retain all attempted deploy-time alerts
	DefaultAttemptedDeployAlertRetention = 7
	// DefaultAttemptedRuntimeAlertRetention is the number of days to retain all attempted run-time alerts
	DefaultAttemptedRuntimeAlertRetention = 7
	// DefaultExpiredVulnReqRetention is the number of days to retain expired vulnerability requests.
	DefaultExpiredVulnReqRetention = 90
	// DefaultDecommissionedClusterRetentionDays is the number of days to retain a cluster that is unreachable.
	DefaultDecommissionedClusterRetentionDays = 0
	// DefaultReportHistoryRetentionWindow number of days to retain reports
	DefaultReportHistoryRetentionWindow = 7
	// DefaultDownloadableReportRetentionDays number of days to retain downloadable reports
	DefaultDownloadableReportRetentionDays = 7
	// DefaultDownloadableReportGlobalRetentionBytes is the maximum total upper limit in bytes for all downloadable reports
	DefaultDownloadableReportGlobalRetentionBytes = 500 * 1024 * 1024
)

var (
	once sync.Once

	d DataStore

	defaultPrivateConfig = storage.PrivateConfig{
		ImageRetentionDurationDays: DefaultImageRetention,
		AlertRetention: &storage.PrivateConfig_AlertConfig{
			AlertConfig: &storage.AlertRetentionConfig{
				ResolvedDeployRetentionDurationDays:   DefaultDeployAlertRetention,
				DeletedRuntimeRetentionDurationDays:   DefaultDeletedRuntimeAlertRetention,
				AllRuntimeRetentionDurationDays:       DefaultRuntimeAlertRetention,
				AttemptedDeployRetentionDurationDays:  DefaultAttemptedDeployAlertRetention,
				AttemptedRuntimeRetentionDurationDays: DefaultAttemptedRuntimeAlertRetention,
			},
		},
		ExpiredVulnReqRetentionDurationDays: DefaultExpiredVulnReqRetention,
	}
)

func initialize() {
	store := pgStore.New(globaldb.GetPostgres())

	d = New(store)

	ctx := sac.WithGlobalAccessScopeChecker(
		context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))
	config, err := d.GetConfig(ctx)
	if err != nil {
		panic(err)
	}

	privateConfig := config.GetPrivateConfig()
	needsUpsert := false
	if privateConfig == nil {
		privateConfig = &defaultPrivateConfig
		needsUpsert = true
	}

	if privateConfig.GetDecommissionedClusterRetention() == nil {
		privateConfig.DecommissionedClusterRetention = &storage.DecommissionedClusterRetentionConfig{
			RetentionDurationDays: DefaultDecommissionedClusterRetentionDays,
		}
		needsUpsert = true
	}

	if privateConfig.GetReportRetentionConfig() == nil {
		privateConfig.ReportRetentionConfig = &storage.ReportRetentionConfig{
			HistoryRetentionDurationDays:           DefaultReportHistoryRetentionWindow,
			DownloadableReportRetentionDays:        DefaultDownloadableReportRetentionDays,
			DownloadableReportGlobalRetentionBytes: DefaultDownloadableReportGlobalRetentionBytes,
		}
		needsUpsert = true
	}

	if needsUpsert {
		utils.Must(d.UpsertConfig(ctx, &storage.Config{
			PublicConfig:  config.GetPublicConfig(),
			PrivateConfig: privateConfig,
		}))
	}
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return d
}
