package datastore

import (
	"context"

	"github.com/stackrox/rox/central/apitoken/datastore/internal/store"
	postgresStore "github.com/stackrox/rox/central/apitoken/datastore/internal/store/postgres"
	rocksdbStore "github.com/stackrox/rox/central/apitoken/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	integrationSAC = sac.ForResource(resources.Integration)

	log = logging.LoggerForModule()
)

type datastoreImpl struct {
	storage  store.Store
	searcher search.Searcher

	sync.Mutex
}

func newPostgres(pool *postgres.DB) *datastoreImpl {
	storage := postgresStore.New(pool)
	indexer := postgresStore.NewIndexer(pool)
	searcher := blevesearch.WrapUnsafeSearcherAsSearcher(indexer)

	return &datastoreImpl{
		storage:  storage,
		searcher: searcher,
	}
}

func newRocks(rocksDBInstance *rocksdb.RocksDB) *datastoreImpl {
	storage := rocksdbStore.New(rocksDBInstance)
	return &datastoreImpl{
		storage:  storage,
		searcher: nil,
	}
}

func (b *datastoreImpl) AddToken(ctx context.Context, token *storage.TokenMetadata) error {
	if ok, err := integrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	b.Lock()
	defer b.Unlock()

	return b.storage.Upsert(ctx, token)
}

func (b *datastoreImpl) GetTokenOrNil(ctx context.Context, id string) (token *storage.TokenMetadata, err error) {
	if ok, err := integrationSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	b.Lock()
	defer b.Unlock()

	token, exists, err := b.storage.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	return token, nil
}

func (b *datastoreImpl) GetTokens(ctx context.Context, req *v1.GetAPITokensRequest) ([]*storage.TokenMetadata, error) {
	if ok, err := integrationSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	b.Lock()
	defer b.Unlock()

	var tokens []*storage.TokenMetadata
	walkFn := func() error {
		tokens = tokens[:0]
		return b.storage.Walk(ctx, func(token *storage.TokenMetadata) error {
			if req.GetRevokedOneof() != nil && req.GetRevoked() != token.GetRevoked() {
				return nil
			}
			tokens = append(tokens, token)
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(walkFn); err != nil {
		return nil, err
	}
	return tokens, nil
}

func (b *datastoreImpl) RevokeToken(ctx context.Context, id string) (bool, error) {
	if ok, err := integrationSAC.WriteAllowed(ctx); err != nil {
		return false, err
	} else if !ok {
		return false, sac.ErrResourceAccessDenied
	}

	b.Lock()
	defer b.Unlock()

	token, exists, err := b.storage.Get(ctx, id)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}
	token.Revoked = true

	if err := b.storage.Upsert(ctx, token); err != nil {
		return false, err
	}
	return true, nil
}

func (b *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	if err := sac.VerifyAuthzOK(integrationSAC.ReadAllowed(ctx)); err != nil {
		return nil, err
	}
	return b.searcher.Search(ctx, q)
}

func (b *datastoreImpl) SearchRawTokens(ctx context.Context, q *v1.Query) ([]*storage.TokenMetadata, error) {
	if err := sac.VerifyAuthzOK(integrationSAC.ReadAllowed(ctx)); err != nil {
		return nil, err
	}
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return b.storage.GetByQuery(ctx, q)
	}
	results, err := b.searcher.Search(ctx, q)
	if err != nil {
		return nil, err
	}
	objects := make([]*storage.TokenMetadata, 0, len(results))
	for _, r := range results {
		token, found, err := b.storage.Get(ctx, r.ID)
		if err != nil {
			return nil, err
		}
		if found {
			objects = append(objects, token)
		}
	}
	return objects, nil
}
