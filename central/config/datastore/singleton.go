package datastore

import (
	"context"

	"github.com/stackrox/rox/central/config/store"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/role/resources"
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
	d = New(store.New(globaldb.GetGlobalDB()))

	ctx := sac.WithGlobalAccessScopeChecker(
		context.Background(),
		sac.OneStepSCC{
			sac.AccessModeScopeKey(storage.Access_READ_ACCESS): sac.AllowFixedScopes(
				sac.ResourceScopeKeys(resources.Config)),
			sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS): sac.AllowFixedScopes(
				sac.ResourceScopeKeys(resources.Config)),
		})
	config, err := d.GetConfig(ctx)
	if err != nil {
		panic(err)
	}

	if config.GetPrivateConfig() == nil {
		utils.Must(d.UpsertConfig(ctx, &storage.Config{
			PublicConfig:  config.GetPublicConfig(),
			PrivateConfig: &defaultPrivateConfig,
		}))
	}
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return d
}
