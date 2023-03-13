package m172tom173

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/frozenschema/v73"
	newAPITokenStore "github.com/stackrox/rox/migrator/migrations/m_172_to_m_173_enable_search_on_api_tokens/newapitokenpostgresstore"
	oldAPITokenStore "github.com/stackrox/rox/migrator/migrations/m_172_to_m_173_enable_search_on_api_tokens/oldapitokenpostgresstore"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"gorm.io/gorm"
)

const (
	batchSize = 500

	startSeqNum = 172
)

var (
	migration = types.Migration{
		StartingSeqNum: startSeqNum,
		VersionAfter:   &storage.Version{SeqNum: int32(startSeqNum + 1)},
		Run: func(database *types.Databases) error {
			// Migration code comes here
			return nil
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

// Additional code to support the migration

func migrateAPITokens(postgresDB *postgres.DB, gormDB *gorm.DB) error {
	ctx := context.Background()
	oldStore := oldAPITokenStore.New(postgresDB)
	newStore := newAPITokenStore.New(postgresDB)
	pgutils.CreateTableFromModel(ctx, gormDB, frozenSchema.CreateTableAPITokensStmt)

	tokensToUpsert := make([]*storage.TokenMetadata, 0, batchSize)
	err := oldStore.Walk(ctx, func(obj *storage.TokenMetadata) error {
		tokensToUpsert = append(tokensToUpsert, obj)
		if len(tokensToUpsert) >= batchSize {
			upsertErr := newStore.UpsertMany(ctx, tokensToUpsert)
			if upsertErr != nil {
				return upsertErr
			}
			tokensToUpsert = tokensToUpsert[:0]
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(tokensToUpsert) > 0 {
		return newStore.UpsertMany(ctx, tokensToUpsert)
	}
	return nil
}
