package edgepostgresstorefortest

import (
	"context"
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/jackc/pgx/v5"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/policymigrationhelper/edgepostgresstorefortest/schema"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

var (
	schema = frozenSchema.PolicyCategoryEdgesSchema
)

type storeType = storage.PolicyCategoryEdge

// Store is the interface to interact with the storage for storage.PolicyCategoryEdge
type Store interface {
	Upsert(ctx context.Context, obj *storeType) error
	Delete(ctx context.Context, id string) error
	DeleteByQuery(ctx context.Context, q *v1.Query) ([]string, error)
}

type storeImpl struct {
	db postgres.DB
}

func (s *storeImpl) Upsert(ctx context.Context, obj *storeType) error {
	return pgutils.Retry(func() error {
		return s.upsert(ctx, obj)
	})
}

// Delete removes the object associated to the specified ID from the store.
func (s *storeImpl) Delete(ctx context.Context, id string) error {
	var sacQueryFilter *v1.Query

	q := search.ConjunctionQuery(
		sacQueryFilter,
		search.NewQueryBuilder().AddDocIDs(id).ProtoQuery(),
	)

	return pgSearch.RunDeleteRequestForSchema(ctx, schema, q, s.db)
}

// DeleteByQuery removes the objects from the store based on the passed query.
func (s *storeImpl) DeleteByQuery(ctx context.Context, q *v1.Query) ([]string, error) {
	var sacQueryFilter *v1.Query

	query := search.ConjunctionQuery(
		sacQueryFilter,
		q,
	)

	return pgSearch.RunDeleteRequestReturningIDsForSchema(ctx, schema, query, s.db)
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

func (s *storeImpl) upsert(ctx context.Context, objs ...*storage.PolicyCategoryEdge) error {
	conn, release, err := s.acquireConn(ctx, ops.Get, "PolicyCategoryEdge")
	if err != nil {
		return err
	}
	defer release()

	for _, obj := range objs {
		batch := &pgx.Batch{}
		if err := insertIntoPolicyCategoryEdges(batch, obj); err != nil {
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

func insertIntoPolicyCategoryEdges(batch *pgx.Batch, obj *storage.PolicyCategoryEdge) error {

	serialized, marshalErr := obj.Marshal()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		// parent primary keys start
		obj.GetId(),
		obj.GetPolicyId(),
		obj.GetCategoryId(),
		serialized,
	}

	finalStr := "INSERT INTO policy_category_edges (Id, PolicyId, CategoryId, serialized) VALUES($1, $2, $3, $4) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, PolicyId = EXCLUDED.PolicyId, CategoryId = EXCLUDED.CategoryId, serialized = EXCLUDED.serialized"
	batch.Queue(finalStr, values...)

	return nil
}

// endregion Used for testing
