package datastore

import (
	"context"

	configStore "github.com/stackrox/rox/central/config/store"
	"github.com/stackrox/rox/central/config/store/bolt"
	"github.com/stackrox/rox/central/config/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
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
	var store configStore.Store
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		store = postgres.New(globaldb.GetPostgres())
	} else {
		store = bolt.New(globaldb.GetGlobalDB())
	}
	d = New(store)

	ctx := sac.WithGlobalAccessScopeChecker(
		context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Config)))
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

	if features.DecommissionedClusterRetention.Enabled() && privateConfig.GetDecommissionedClusterRetention() == nil {
		privateConfig.DecommissionedClusterRetention = &storage.DecommissionedClusterRetentionConfig{
			RetentionDurationDays: DefaultDecommissionedClusterRetentionDays,
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
