package m174tom175

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	newAPITokenStore "github.com/stackrox/rox/migrator/migrations/m_174_to_m_175_enable_search_on_api_tokens/newapitokenpostgresstore"
	oldAPITokenStore "github.com/stackrox/rox/migrator/migrations/m_174_to_m_175_enable_search_on_api_tokens/oldapitokenpostgresstore"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/m_174_to_m_175_enable_search_on_api_tokens/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"gorm.io/gorm"
)

const (
	batchSize = 500

	startSeqNum = 174
)

var (
	migration = types.Migration{
		StartingSeqNum: startSeqNum,
		VersionAfter:   &storage.Version{SeqNum: int32(startSeqNum + 1)},
		Run: func(database *types.Databases) error {
			return migrateAPITokens(database.PostgresDB, database.GormDB)
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func migrateAPITokens(postgresDB postgres.DB, gormDB *gorm.DB) error {
	ctx := sac.WithAllAccess(context.Background())
	oldStore := oldAPITokenStore.New(postgresDB)
	pgutils.CreateTableFromModel(ctx, gormDB, frozenSchema.CreateTableAPITokensStmt)
	newStore := newAPITokenStore.New(postgresDB)

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
