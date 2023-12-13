package n42ton43

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/frozenschema/v73"
	"github.com/stackrox/rox/migrator/migrations/loghelper"
	legacy "github.com/stackrox/rox/migrator/migrations/n_42_to_n_43_postgres_process_indicators/legacy"
	pgStore "github.com/stackrox/rox/migrator/migrations/n_42_to_n_43_postgres_process_indicators/postgres"
	"github.com/stackrox/rox/migrator/types"
	pkgMigrations "github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/process/normalize"
	"gorm.io/gorm"
)

var (
	startingSeqNum = pkgMigrations.BasePostgresDBVersionSeqNum() + 42 // 153

	migration = types.Migration{
		StartingSeqNum: startingSeqNum,
		VersionAfter:   &storage.Version{SeqNum: int32(startingSeqNum + 1)}, // 154
		Run: func(databases *types.Databases) error {
			legacyStore, err := legacy.New(databases.PkgRocksDB)
			if err != nil {
				return err
			}
			if err := move(databases.DBCtx, databases.GormDB, databases.PostgresDB, legacyStore); err != nil {
				return errors.Wrap(err,
					"moving process_indicators from rocksdb to postgres")
			}
			return nil
		},
	}
	batchSize = 10000
	log       = loghelper.LogWrapper{}
)

func move(ctx context.Context, gormDB *gorm.DB, postgresDB postgres.DB, legacyStore legacy.Store) error {
	store := pgStore.New(postgresDB)
	pgutils.CreateTableFromModel(context.Background(), gormDB, frozenSchema.CreateTableProcessIndicatorsStmt)

	var processIndicators []*storage.ProcessIndicator
	err := walk(ctx, legacyStore, func(obj *storage.ProcessIndicator) error {
		normalize.Indicator(obj)
		processIndicators = append(processIndicators, obj)
		if len(processIndicators) == batchSize {
			if err := store.UpsertMany(ctx, processIndicators); err != nil {
				log.WriteToStderrf("failed to persist process_indicators to store %v", err)
				return err
			}
			processIndicators = processIndicators[:0]
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(processIndicators) > 0 {
		if err = store.UpsertMany(ctx, processIndicators); err != nil {
			log.WriteToStderrf("failed to persist process_indicators to store %v", err)
			return err
		}
	}
	return nil
}

func walk(ctx context.Context, s legacy.Store, fn func(obj *storage.ProcessIndicator) error) error {
	return s.Walk(ctx, fn)
}

func init() {
	migrations.MustRegisterMigration(migration)
}
