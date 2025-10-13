package categorypostgresstorefortest

import (
	"context"
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/jackc/pgx/v5"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/policymigrationhelper/categorypostgresstorefortest/schema"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

var (
	schema = frozenSchema.PolicyCategoriesSchema
)

type storeType = storage.PolicyCategory

// Store is the interface to interact with the storage for storage.PolicyCategory
type Store interface {
	Upsert(ctx context.Context, obj *storeType) error
	GetAll(ctx context.Context) ([]*storeType, error)
}

type storeImpl struct {
	db postgres.DB
}

func (s *storeImpl) Upsert(ctx context.Context, obj *storeType) error {
	return pgutils.Retry(ctx, func() error {
		return s.upsert(ctx, obj)
	})
}

func (s *storeImpl) GetAll(ctx context.Context) ([]*storeType, error) {
	var objs []*storeType
	err := s.Walk(ctx, func(obj *storeType) error {
		objs = append(objs, obj)
		return nil
	})
	return objs, err
}

// Walk iterates through each policy category
func (s *storeImpl) Walk(ctx context.Context, fn func(obj *storage.PolicyCategory) error) error {
	var sacQueryFilter *v1.Query
	return pgSearch.RunCursorQueryForSchemaFn(ctx, schema, sacQueryFilter, s.db, fn)
}

// New returns a new Store instance using the provided sql instance.
// Only used for tests
func New(db postgres.DB, _ *testing.T) Store {
	return &storeImpl{
		db: db,
	}
}

// region Helper functions

func (s *storeImpl) acquireConn(ctx context.Context, _ ops.Op, _ string) (*postgres.Conn, func(), error) {
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, nil, err
	}
	return conn, conn.Release, nil
}

func (s *storeImpl) upsert(ctx context.Context, objs ...*storage.PolicyCategory) error {
	conn, release, err := s.acquireConn(ctx, ops.Get, "PolicyCategory")
	if err != nil {
		return err
	}
	defer release()

	for _, obj := range objs {
		batch := &pgx.Batch{}
		if err := insertIntoPolicyCategories(batch, obj); err != nil {
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

func insertIntoPolicyCategories(batch *pgx.Batch, obj *storage.PolicyCategory) error {

	serialized, marshalErr := obj.MarshalVT()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		// parent primary keys start
		obj.GetId(),
		obj.GetName(),
		serialized,
	}

	finalStr := "INSERT INTO policy_categories (Id, Name, serialized) VALUES($1, $2, $3) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, Name = EXCLUDED.Name, serialized = EXCLUDED.serialized"
	batch.Queue(finalStr, values...)

	return nil
}

// endregion Helper functions
