package postgres

import (
	"context"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/sync"
	"gorm.io/gorm"
)

const (
	baseTable = "policy_categories"

	cursorBatchSize = 50
)

var (
	log            = logging.LoggerForModule()
	schema         = pkgSchema.PolicyCategoriesSchema
	targetResource = resources.Policy
)

// Store is the interface to interact with the storage for storage.PolicyCategory
type Store interface {
	Count(ctx context.Context) (int, error)
	Walk(ctx context.Context, fn func(obj *storage.PolicyCategory) error) error

	Upsert(ctx context.Context, obj *storage.PolicyCategory) error
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.PolicyCategory, error)
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

// GetByQuery returns the objects from the store matching the query.
func (s *storeImpl) GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.PolicyCategory, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetByQuery, "PolicyCategory")

	var sacQueryFilter *v1.Query

	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_ACCESS).Resource(targetResource)
	if !scopeChecker.IsAllowed() {
		return nil, nil
	}
	q := search.ConjunctionQuery(
		sacQueryFilter,
		query,
	)

	rows, err := postgres.RunGetManyQueryForSchema[storage.PolicyCategory](ctx, schema, q, s.db)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return rows, nil
}

// Count returns the number of objects in the store.
func (s *storeImpl) Count(ctx context.Context) (int, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Count, "PolicyCategory")

	var sacQueryFilter *v1.Query

	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_ACCESS).Resource(targetResource)
	if !scopeChecker.IsAllowed() {
		return 0, nil
	}

	return postgres.RunCountRequestForSchema(ctx, schema, sacQueryFilter, s.db)
}

func (s *storeImpl) Walk(ctx context.Context, fn func(obj *storage.PolicyCategory) error) error {
	var sacQueryFilter *v1.Query
	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_ACCESS).Resource(targetResource)
	if !scopeChecker.IsAllowed() {
		return nil
	}
	fetcher, closer, err := postgres.RunCursorQueryForSchema[storage.PolicyCategory](ctx, schema, sacQueryFilter, s.db)
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

// Upsert saves the current state of an object in storage.
func (s *storeImpl) Upsert(ctx context.Context, obj *storage.PolicyCategory) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Upsert, "PolicyCategory")

	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_WRITE_ACCESS).Resource(targetResource)
	if !scopeChecker.IsAllowed() {
		return sac.ErrResourceAccessDenied
	}

	return pgutils.Retry(func() error {
		return s.upsert(ctx, obj)
	})
}

//// Interface functions - END

//// Used for testing

// CreateTableAndNewStore returns a new Store instance for testing.
func CreateTableAndNewStore(ctx context.Context, db *pgxpool.Pool, gormDB *gorm.DB) Store {
	pkgSchema.ApplySchemaForTable(ctx, gormDB, baseTable)
	return New(db)
}

// Destroy drops the tables associated with the target object type.
func Destroy(ctx context.Context, db *pgxpool.Pool) {
	dropTablePolicyCategories(ctx, db)
}

func dropTablePolicyCategories(ctx context.Context, db *pgxpool.Pool) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS policy_categories CASCADE")

}

func (s *storeImpl) acquireConn(ctx context.Context, op ops.Op, typ string) (*pgxpool.Conn, func(), error) {
	defer metrics.SetAcquireDBConnDuration(time.Now(), op, typ)
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
		if err := insertIntoPolicyCategories(ctx, batch, obj); err != nil {
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

func insertIntoPolicyCategories(ctx context.Context, batch *pgx.Batch, obj *storage.PolicyCategory) error {

	serialized, marshalErr := obj.Marshal()
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

//// Used for testing - END
