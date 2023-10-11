package m192tom193

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
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
	// Create name column
	pgutils.CreateTableFromModel(database.DBCtx, database.GormDB, midPkgSchema.CreateTableAPITokensStmt)
	migratedTokens := make([]*storage.TokenMetadata, 0, batchSize)
	walkErr := oldAPITokenStorage.Walk(database.DBCtx, func(obj *storage.TokenMetadata) error {
		seenTokenNames[obj.GetName()]++
		migratedObj := obj.Clone()
		migratedName := getNewTokenName(obj.GetName())
		migratedObj.Name = migratedName
		migratedTokens = append(migratedTokens, migratedObj)
		if len(migratedTokens) >= batchSize {
			upsertErr := upsertBatch(database.DBCtx, newAPITokenStorage, migratedTokens)
			migratedTokens = migratedTokens[:0]
			return upsertErr
		}
		return nil
	})
	if len(migratedTokens) > 0 {
		upsertErr := upsertBatch(database.DBCtx, newAPITokenStorage, migratedTokens)
		if upsertErr != nil {
			return upsertErr
		}
	}
	pgutils.CreateTableFromModel(database.DBCtx, database.GormDB, newPkgSchema.CreateTableAPITokensStmt)

	return walkErr
}

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
	storage newAPITokenStore.Store,
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
