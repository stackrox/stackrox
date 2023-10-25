package m192tom193

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	APITokenStore "github.com/stackrox/rox/migrator/migrations/m_192_to_m_193_make_api_token_names_unique/apitokenstore"
	newAPITokenStore "github.com/stackrox/rox/migrator/migrations/m_192_to_m_193_make_api_token_names_unique/apitokenstore/new"
	oldAPITokenStore "github.com/stackrox/rox/migrator/migrations/m_192_to_m_193_make_api_token_names_unique/apitokenstore/old"
	midPkgSchema "github.com/stackrox/rox/migrator/migrations/m_192_to_m_193_make_api_token_names_unique/schema/mid"
	newPkgSchema "github.com/stackrox/rox/migrator/migrations/m_192_to_m_193_make_api_token_names_unique/schema/new"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
)

const (
	batchSize = 500
)

var (
	seenTokenNames = make(map[string]int)
)

func migrate(database *types.Databases) error {
	oldAPITokenStorage := oldAPITokenStore.New(database.PostgresDB)
	newAPITokenStorage := newAPITokenStore.New(database.PostgresDB)
	migrator := newMigrator(oldAPITokenStorage, newAPITokenStorage)
	return migrator.doMigrate(database)
}

type migratorImpl struct {
	oldStore APITokenStore.Store
	newStore APITokenStore.Store
}

func newMigrator(oldStore APITokenStore.Store, newStore APITokenStore.Store) *migratorImpl {
	return &migratorImpl{
		oldStore: oldStore,
		newStore: newStore,
	}
}

func (m *migratorImpl) doMigrate(database *types.Databases) error {
	// Create name column.
	pgutils.CreateTableFromModel(database.DBCtx, database.GormDB, midPkgSchema.CreateTableAPITokensStmt)
	// All tokens must be written back to the DB using the new store in order
	// to populate the `Name` column. The call to getNewTokenName should
	// ensure that all names in the table are unique.
	migratedTokens := make([]*storage.TokenMetadata, 0, batchSize)
	walkErr := m.oldStore.Walk(database.DBCtx, func(obj *storage.TokenMetadata) error {
		seenTokenNames[obj.GetName()]++
		migratedObj := obj.Clone()
		migratedName := getNewTokenName(obj.GetName())
		migratedObj.Name = migratedName
		migratedTokens = append(migratedTokens, migratedObj)
		if len(migratedTokens) >= batchSize {
			upsertErr := upsertBatch(database.DBCtx, m.newStore, migratedTokens)
			migratedTokens = migratedTokens[:0]
			return upsertErr
		}
		return nil
	})
	if len(migratedTokens) > 0 {
		upsertErr := upsertBatch(database.DBCtx, m.newStore, migratedTokens)
		if upsertErr != nil {
			return upsertErr
		}
	}
	// Create unique constraint on name column.
	pgutils.CreateTableFromModel(database.DBCtx, database.GormDB, newPkgSchema.CreateTableAPITokensStmt)

	return walkErr
}

// getNewTokenName returns a unique name for the current token name, based on
// the names that were seen at that point of the migration.
// - If the name is seen for the first time (count is 1), then it is used
// as-is.
// - If the name was already seen at least once (count is strictly greater
// than 1), then it is suffixed with the current name occurrence count (2 for
// the second, 3 for the third...) and the new name is also counted to avoid
// collisions with names of tokens not processed yet.
func getNewTokenName(tokenName string) string {
	if seenTokenNames[tokenName] <= 1 {
		return tokenName
	}
	migratedName := fmt.Sprintf("%s (%d)", tokenName, seenTokenNames[tokenName])
	for seenTokenNames[migratedName] > 0 {
		seenTokenNames[tokenName]++
		migratedName = fmt.Sprintf("%s (%d)", tokenName, seenTokenNames[tokenName])
	}
	seenTokenNames[migratedName]++
	return migratedName
}

func upsertBatch(
	ctx context.Context,
	storage APITokenStore.Store,
	batch []*storage.TokenMetadata,
) error {
	upsertErr := storage.UpsertMany(ctx, batch)
	if upsertErr != nil {
		tokenIDs := make([]string, 0, len(batch))
		for _, t := range batch {
			tokenIDs = append(tokenIDs, t.GetId())
		}
		return errors.Wrap(
			upsertErr,
			fmt.Sprintf("failed to update tokens %q", strings.Join(tokenIDs, ",")),
		)
	}
	return nil
}
