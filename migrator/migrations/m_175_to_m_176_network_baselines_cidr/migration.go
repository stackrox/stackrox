package m_175_to_m_176_network_baselines_cidr

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/m_175_to_m_176_network_baselines_cidr/networkbaselinestore"
	"github.com/stackrox/rox/migrator/migrations/m_175_to_m_176_network_baselines_cidr/networkentitystore"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
	"gorm.io/gorm"
)

const (
	startSeqNum = 175

	batchSize = 500
)

var (
	migration = types.Migration{
		StartingSeqNum: startSeqNum,
		VersionAfter:   &storage.Version{SeqNum: int32(startSeqNum + 1)},
		Run: func(database *types.Databases) error {
			return addCIDRBlockToBaselines(database.PostgresDB, database.GormDB)
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func addCIDRBlockToBaselines(postgresDB *postgres.DB, gormDB *gorm.DB) error {
	// Initialize the copy versions of baseline store and
	ctx := context.Background()
	networkBaselineStore := networkbaselinestore.CreateTableAndNewStore(ctx, postgresDB, gormDB)
	networkEntityStore := networkentitystore.CreateTableAndNewStore(ctx, postgresDB, gormDB)

	baselinesToUpsert := make([]*storage.NetworkBaseline, 0, batchSize)
	err := networkBaselineStore.Walk(ctx, func(baseline *storage.NetworkBaseline) error {
		updateBaseline := false
		for idx, peer := range baseline.GetPeers() {
			info := peer.GetEntity().GetInfo()
			externalSource := info.GetExternalSource()
			if externalSource != nil {
				// Peer is an external entity and needs to be updated with CIDR block
				entity, exists, err := networkEntityStore.Get(ctx, info.GetId())
				if err != nil {
					return err
				}

				if !exists {
					return errors.Wrapf(err, "no network entity for peer %s in baseline for deployment %s",
						info.GetId(),
						baseline.GetDeploymentId())
				}

				if entity.GetInfo().GetExternalSource() == nil {
					return errors.Wrapf(err, "inconsistent type for peer %s in baseline for deployment %s: expecting EXTERNAL_SOURCE but is %s",
						info.GetId(),
						baseline.GetDeploymentId(),
						entity.GetInfo().GetType())
				}

				entityCidrBlock := entity.GetInfo().GetExternalSource().GetCidr()
				externalSource.Source = &storage.NetworkEntityInfo_ExternalSource_Cidr{
					Cidr: entityCidrBlock,
				}

				// Update peer
				peer.Entity.Info.Desc = &storage.NetworkEntityInfo_ExternalSource_{
					ExternalSource: externalSource,
				}
				baseline.Peers[idx] = peer
				updateBaseline = true
			}
		}

		if updateBaseline {
			baselinesToUpsert = append(baselinesToUpsert, baseline)
		}

		if len(baselinesToUpsert) >= batchSize {
			upsertErr := networkBaselineStore.UpsertMany(ctx, baselinesToUpsert)
			if upsertErr != nil {
				return upsertErr
			}
			baselinesToUpsert = baselinesToUpsert[:0]
		}
		return nil
	})

	if err != nil {
		return err
	}

	if len(baselinesToUpsert) > 0 {
		return networkBaselineStore.UpsertMany(ctx, baselinesToUpsert)
	}

	return nil
}
