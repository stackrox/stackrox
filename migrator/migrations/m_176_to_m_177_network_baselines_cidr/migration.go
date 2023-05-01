package m176tom177

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/m_176_to_m_177_network_baselines_cidr/networkbaselinestore"
	"github.com/stackrox/rox/migrator/migrations/m_176_to_m_177_network_baselines_cidr/networkentitystore"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
)

const (
	startSeqNum = 176

	batchSize = 500
)

var (
	migration = types.Migration{
		StartingSeqNum: startSeqNum,
		VersionAfter:   &storage.Version{SeqNum: int32(startSeqNum + 1)}, // 177
		Run: func(database *types.Databases) error {
			return addCIDRBlockToBaselines(database.PostgresDB)
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func updatePeer(
	ctx context.Context,
	networkEntityStore networkentitystore.Store,
	deploymentID string,
	peer *storage.NetworkBaselinePeer,
) (bool, error) {
	info := peer.GetEntity().GetInfo()
	externalSource := info.GetExternalSource()
	if externalSource != nil {
		// Peer is an external entity and needs to be updated with CIDR block.
		// We need to query data from the Network Entity store to fetch the CIDR block information
		// using peer ID (i.e. external source ID) and append to the Network Entity property in
		// the baseline.
		entity, exists, err := networkEntityStore.Get(ctx, info.GetId())
		if err != nil {
			return false, err
		}

		if !exists {
			return false, errors.Wrapf(err, "no network entity for peer %s in baseline for deployment %s",
				info.GetId(),
				deploymentID)
		}

		if entity.GetInfo().GetExternalSource() == nil {
			return false, errors.Wrapf(err, "inconsistent type for peer %s in baseline for deployment %s: expecting EXTERNAL_SOURCE but is %s",
				info.GetId(),
				deploymentID,
				entity.GetInfo().GetType())
		}

		entityCidrBlock := entity.GetInfo().GetExternalSource().GetCidr()
		externalSource.Source = &storage.NetworkEntityInfo_ExternalSource_Cidr{
			Cidr: entityCidrBlock,
		}

		peer.Entity.Info.Desc = &storage.NetworkEntityInfo_ExternalSource_{
			ExternalSource: externalSource,
		}
		return true, nil
	}
	return false, nil
}

func addCIDRBlockToBaselines(postgresDB postgres.DB) error {
	ctx := context.Background()
	networkBaselineStore := networkbaselinestore.New(postgresDB)
	networkEntityStore := networkentitystore.New(postgresDB)

	baselinesToUpsert := make([]*storage.NetworkBaseline, 0, batchSize)
	err := networkBaselineStore.Walk(ctx, func(baseline *storage.NetworkBaseline) error {
		updateBaseline := false

		// Baseline maintains peers both in `Peers` and `ForbiddenPeers`. They have the
		// same structure. Therefore, both could have External Sources linked.

		for _, peer := range baseline.GetPeers() {
			updated, err := updatePeer(ctx, networkEntityStore, baseline.GetDeploymentId(), peer)
			if err != nil {
				return err
			}

			if updated {
				updateBaseline = true
			}
		}

		for _, peer := range baseline.GetForbiddenPeers() {
			updated, err := updatePeer(ctx, networkEntityStore, baseline.GetDeploymentId(), peer)
			if err != nil {
				return err
			}

			if updated {
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
