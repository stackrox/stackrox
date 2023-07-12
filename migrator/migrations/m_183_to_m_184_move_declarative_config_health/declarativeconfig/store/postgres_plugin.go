package store

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/jackc/pgx/v5"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	migrationSchema "github.com/stackrox/rox/migrator/migrations/m_183_to_m_184_move_declarative_config_health/declarativeconfig/schema"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

// Store is the interface to the config health data layer
type Store interface {
	Get(ctx context.Context, id string) (*storage.DeclarativeConfigHealth, bool, error)
	Upsert(ctx context.Context, obj *storage.DeclarativeConfigHealth) error
	Delete(ctx context.Context, id string) error
	Walk(ctx context.Context, fn func(obj *storage.DeclarativeConfigHealth) error) error
}

const (
	cursorBatchSize = 50
)

var (
	log    = logging.LoggerForModule()
	schema = migrationSchema.DeclarativeConfigHealthsSchema
)

type storeImpl struct {
	db    postgres.DB
	mutex sync.RWMutex
}

// New returns a new Store instance using the provided sql instance.
func New(db postgres.DB) Store {
	return &storeImpl{
		db: db,
	}
}

//// Helper functions

func insertIntoDeclarativeConfigHealths(_ context.Context, batch *pgx.Batch, obj *storage.DeclarativeConfigHealth) error {
	serialized, marshalErr := obj.Marshal()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		// parent primary keys start
		obj.GetId(),
		serialized,
	}

	finalStr := "INSERT INTO declarative_config_healths (Id, serialized) VALUES($1, $2) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, serialized = EXCLUDED.serialized"
	batch.Queue(finalStr, values...)

	return nil
}

func (s *storeImpl) acquireConn(ctx context.Context, _ ops.Op, _ string) (*postgres.Conn, func(), error) {
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, nil, err
	}
	return conn, conn.Release, nil
}

func (s *storeImpl) upsert(ctx context.Context, objs ...*storage.DeclarativeConfigHealth) error {
	conn, release, err := s.acquireConn(ctx, ops.Get, "DeclarativeConfigHealth")
	if err != nil {
		return err
	}
	defer release()

	for _, obj := range objs {
		batch := &pgx.Batch{}
		if err := insertIntoDeclarativeConfigHealths(ctx, batch, obj); err != nil {
			return err
		}
		batchResults := conn.SendBatch(ctx, batch)
		var result *multierror.Error
		for i := 0; i < batch.Len(); i++ {
			_, err := batchResults.Exec()
			result = multierror.Append(result, err)
		}
		if err := batchResults.Close(); err != nil {
			return err
		}
		if err := result.ErrorOrNil(); err != nil {
			return err
		}
	}
	return nil
}

//// Helper functions - END

//// Interface functions

// Upsert saves the current state of an object in storage.
func (s *storeImpl) Upsert(ctx context.Context, obj *storage.DeclarativeConfigHealth) error {
	return pgutils.Retry(func() error {
		return s.upsert(ctx, obj)
	})
}

// Delete removes the object associated to the specified ID from the store.
func (s *storeImpl) Delete(ctx context.Context, id string) error {
	q := search.NewQueryBuilder().AddDocIDs(id).ProtoQuery()
	return pgSearch.RunDeleteRequestForSchema(ctx, schema, q, s.db)
}

// Get returns the object, if it exists from the store.
func (s *storeImpl) Get(ctx context.Context, id string) (*storage.DeclarativeConfigHealth, bool, error) {
	q := search.NewQueryBuilder().AddDocIDs(id).ProtoQuery()

	data, err := pgSearch.RunGetQueryForSchema[storage.DeclarativeConfigHealth](ctx, schema, q, s.db)
	if err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}

	return data, true, nil
}

// Walk iterates over all of the objects in the store and applies the closure.
func (s *storeImpl) Walk(ctx context.Context, fn func(obj *storage.DeclarativeConfigHealth) error) error {
	var sacQueryFilter *v1.Query
	fetcher, closer, err := pgSearch.RunCursorQueryForSchema[storage.DeclarativeConfigHealth](ctx, schema, sacQueryFilter, s.db)
	if err != nil {
		return err
	}
	defer closer()
	for {
		rows, err := fetcher(cursorBatchSize)
		if err != nil {
			return pgutils.ErrNilIfNoRows(err)
		}
		for _, data := range rows {
			if err := fn(data); err != nil {
				return err
			}
		}
		if len(rows) != cursorBatchSize {
			break
		}
	}
	return nil
}
