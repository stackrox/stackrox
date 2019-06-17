package datastore

import (
	"context"

	"github.com/stackrox/rox/central/config/store"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	// DefaultAlertRetention is the number of days to retain alerts for
	DefaultAlertRetention = 30
	// DefaultImageRetention is the number of days to retain images for
	DefaultImageRetention = 7
)

var (
	once sync.Once

	d DataStore

	defaultPrivateConfig = storage.PrivateConfig{
		ImageRetentionDurationDays: DefaultImageRetention,
	}
)

func initialize() {
	d = New(store.New(globaldb.GetGlobalDB()))

	ctx := sac.WithGlobalAccessScopeChecker(
		context.Background(),
		sac.OneStepSCC{
			sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS): sac.AllowFixedScopes(
				sac.ResourceScopeKeys(resources.Config)),
		})
	config, err := d.GetConfig(ctx)
	if err != nil {
		panic(err)
	}

	if config.GetPrivateConfig() == nil {
		utils.Must(d.UpdateConfig(ctx, &storage.Config{
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
