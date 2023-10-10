package m192tom193

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	apiTokenStore "github.com/stackrox/rox/migrator/migrations/m_192_to_m_193_make_api_token_names_unique/apitokenstore"
	"github.com/stackrox/rox/migrator/types"
)

const (
	batchSize = 500
)

var (
	seenTokenNames = make(map[string]int)
)

func migrate(database *types.Databases) error {
	apiTokenStorage := apiTokenStore.New(database.PostgresDB)
	migratedTokens := make([]*storage.TokenMetadata, 0, batchSize)
	walkErr := apiTokenStorage.Walk(database.DBCtx, func(obj *storage.TokenMetadata) error {
		seenTokenNames[obj.GetName()]++
		seenCount := seenTokenNames[obj.GetName()]
		if seenCount <= 1 {
			// No need to migrate, exit early
			return nil
		}
		migratedObj := obj.Clone()
		migratedName := getNewTokenName(obj.GetName())
		migratedObj.Name = migratedName
		migratedTokens = append(migratedTokens, migratedObj)
		if len(migratedTokens) >= batchSize {
			upsertErr := upsertBatch(database.DBCtx, apiTokenStorage, migratedTokens)
			migratedTokens = migratedTokens[:0]
			return upsertErr
		}
		return nil
	})

	return walkErr
}

func getNewTokenName(tokenName string) string {
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
	storage apiTokenStore.Store,
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
