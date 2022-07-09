package n30ton31

import (
	"context"

	protoTypes "github.com/gogo/protobuf/types"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	legacy "github.com/stackrox/rox/migrator/migrations/n_30_to_n_31_postgres_network_flows/legacy"
	pgStore "github.com/stackrox/rox/migrator/migrations/n_30_to_n_31_postgres_network_flows/postgres"
	"github.com/stackrox/rox/migrator/migrations/n_30_to_n_31_postgres_network_flows/store"
	"github.com/stackrox/rox/migrator/types"
	pkgMigrations "github.com/stackrox/rox/pkg/migrations"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/timestamp"
	"gorm.io/gorm"
)

var (
	migration = types.Migration{
		StartingSeqNum: pkgMigrations.CurrentDBVersionSeqNum() + 30,
		VersionAfter:   storage.Version{SeqNum: int32(pkgMigrations.CurrentDBVersionSeqNum()) + 31},
		Run: func(databases *types.Databases) error {
			legacyStore := legacy.NewClusterStore(databases.PkgRocksDB)
			if err := move(databases.GormDB, databases.PostgresDB, legacyStore); err != nil {
				return errors.Wrap(err,
					"moving network_baselines from rocksdb to postgres")
			}
			return nil
		},
	}
	schema = pkgSchema.NetworkFlowsSchema
)

func move(gormDB *gorm.DB, postgresDB *pgxpool.Pool, legacyStore store.ClusterStore) error {
	ctx := sac.WithAllAccess(context.Background())
	pkgSchema.ApplySchemaForTable(context.Background(), gormDB, schema.Table)

	clusterStore := pgStore.NewClusterStore(postgresDB)

	return walk(ctx, legacyStore, func(clusterID string, ts protoTypes.Timestamp, allFlows []*storage.NetworkFlow) error {
		store, err := clusterStore.CreateFlowStore(ctx, clusterID)
		if err != nil {
			return err
		}
		return store.UpsertFlows(ctx, allFlows, timestamp.FromProtobuf(&ts))
	})
}

func walk(ctx context.Context, s store.ClusterStore, fn func(clusterID string, ts protoTypes.Timestamp, allFlows []*storage.NetworkFlow) error) error {
	return s.Walk(ctx, fn)
}

func init() {
	migrations.MustRegisterMigration(migration)
}
