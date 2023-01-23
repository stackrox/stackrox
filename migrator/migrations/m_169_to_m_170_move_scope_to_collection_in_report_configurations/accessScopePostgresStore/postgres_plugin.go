package postgres

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

// This file is a partial copy of central/resourcecollection/datastore/store/postgres/store.go
// in the state it had when the migration was written.
// Only the relevant functions (Upsert and UpsertMany) are kept.
// The kept functions are stripped from the scoped access control checks.

const (
	baseTable = "simple_access_scopes"

	batchAfter = 100

	// using copyFrom, we may not even want to batch.  It would probably be simpler
	// to deal with failures if we just sent it all.  Something to think about as we
	// proceed and move into more e2e and larger performance testing
	batchSize = 10000

	cursorBatchSize = 50
	deleteBatchSize = 5000
)

var (
	log    = logging.LoggerForModule()
	schema = pkgSchema.RolesSchema
)

// Store is the interface to interact with the storage for storage.SimpleAccessScope
type Store interface {
	Get(ctx context.Context, id string) (*storage.SimpleAccessScope, bool, error)
}

type storeImpl struct {
	db    *pgxpool.Pool
	mutex sync.Mutex
}

// New returns a new Store instance using the provided sql instance.
func New(db *pgxpool.Pool) Store {
	return &storeImpl{
		db: db,
	}
}

// Get returns the object, if it exists from the store.
func (s *storeImpl) Get(ctx context.Context, id string) (*storage.SimpleAccessScope, bool, error) {
	var sacQueryFilter *v1.Query

	q := search.ConjunctionQuery(
		sacQueryFilter,
		search.NewQueryBuilder().AddDocIDs(id).ProtoQuery(),
	)

	data, err := postgres.RunGetQueryForSchema[storage.SimpleAccessScope](ctx, schema, q, s.db)
	if err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}

	return data, true, nil
}
