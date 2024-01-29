package postgres

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/frozenschema/v74"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

// This file is a copy of central/policy/store/postgres/store.go at the time of the migration.
// The only difference between that and this is that SAC checks, a few unused params and imports are removed.
// Unused interface functions are _NOT_ stripped out

const (
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
	schema = frozenSchema.PoliciesSchema
)

// Store is the interface to interact with the storage for storage.Policy
type Store interface {
	Upsert(ctx context.Context, obj *storage.Policy) error
	UpsertMany(ctx context.Context, objs []*storage.Policy) error
	Delete(ctx context.Context, id string) error
	DeleteByQuery(ctx context.Context, q *v1.Query) error
	DeleteMany(ctx context.Context, identifiers []string) error

	Count(ctx context.Context) (int, error)
	Exists(ctx context.Context, id string) (bool, error)

	Get(ctx context.Context, id string) (*storage.Policy, bool, error)
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.Policy, error)
	GetMany(ctx context.Context, identifiers []string) ([]*storage.Policy, []int, error)
	GetIDs(ctx context.Context) ([]string, error)
	GetAll(ctx context.Context) ([]*storage.Policy, error)

	Walk(ctx context.Context, fn func(obj *storage.Policy) error) error

	AckKeysIndexed(ctx context.Context, keys ...string) error
	GetKeysToIndex(ctx context.Context) ([]string, error)
}

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

func insertIntoPolicies(_ context.Context, batch *pgx.Batch, obj *storage.Policy) error {

	serialized, marshalErr := obj.Marshal()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		// parent primary keys start
		obj.GetId(),
		obj.GetName(),
		obj.GetDescription(),
		obj.GetDisabled(),
		obj.GetCategories(),
		obj.GetLifecycleStages(),
		obj.GetSeverity(),
		obj.GetEnforcementActions(),
		pgutils.NilOrTime(obj.GetLastUpdated()),
		obj.GetSORTName(),
		obj.GetSORTLifecycleStage(),
		obj.GetSORTEnforcement(),
		serialized,
	}

	finalStr := "INSERT INTO policies (Id, Name, Description, Disabled, Categories, LifecycleStages, Severity, EnforcementActions, LastUpdated, SORTName, SORTLifecycleStage, SORTEnforcement, serialized) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, Name = EXCLUDED.Name, Description = EXCLUDED.Description, Disabled = EXCLUDED.Disabled, Categories = EXCLUDED.Categories, LifecycleStages = EXCLUDED.LifecycleStages, Severity = EXCLUDED.Severity, EnforcementActions = EXCLUDED.EnforcementActions, LastUpdated = EXCLUDED.LastUpdated, SORTName = EXCLUDED.SORTName, SORTLifecycleStage = EXCLUDED.SORTLifecycleStage, SORTEnforcement = EXCLUDED.SORTEnforcement, serialized = EXCLUDED.serialized"
	batch.Queue(finalStr, values...)

	return nil
}

func (s *storeImpl) copyFromPolicies(ctx context.Context, tx *postgres.Tx, objs ...*storage.Policy) error {

	inputRows := [][]interface{}{}

	var err error

	// This is a copy so first we must delete the rows and re-add them
	// Which is essentially the desired behaviour of an upsert.
	var deletes []string

	copyCols := []string{

		"id",

		"name",

		"description",

		"disabled",

		"categories",

		"lifecyclestages",

		"severity",

		"enforcementactions",

		"lastupdated",

		"sortname",

		"sortlifecyclestage",

		"sortenforcement",

		"serialized",
	}

	for idx, obj := range objs {
		// Todo: ROX-9499 Figure out how to more cleanly template around this issue.
		log.Debugf("This is here for now because there is an issue with pods_TerminatedInstances where the obj "+
			"in the loop is not used as it only consists of the parent ID and the index.  Putting this here as a stop gap "+
			"to simply use the object.  %s", obj)

		serialized, marshalErr := obj.Marshal()
		if marshalErr != nil {
			return marshalErr
		}

		inputRows = append(inputRows, []interface{}{

			obj.GetId(),

			obj.GetName(),

			obj.GetDescription(),

			obj.GetDisabled(),

			obj.GetCategories(),

			obj.GetLifecycleStages(),

			obj.GetSeverity(),

			obj.GetEnforcementActions(),

			pgutils.NilOrTime(obj.GetLastUpdated()),

			obj.GetSORTName(),

			obj.GetSORTLifecycleStage(),

			obj.GetSORTEnforcement(),

			serialized,
		})

		// Add the ID to be deleted.
		deletes = append(deletes, obj.GetId())

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// copy does not upsert so have to delete first.  parent deletion cascades so only need to
			// delete for the top level parent

			if err := s.DeleteMany(ctx, deletes); err != nil {
				return err
			}
			// clear the inserts and vals for the next batch
			deletes = nil

			_, err = tx.CopyFrom(ctx, pgx.Identifier{"policies"}, copyCols, pgx.CopyFromRows(inputRows))

			if err != nil {
				return err
			}

			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	return err
}

func (s *storeImpl) acquireConn(ctx context.Context, _ ops.Op, _ string) (*postgres.Conn, func(), error) {
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, nil, err
	}
	return conn, conn.Release, nil
}

func (s *storeImpl) copyFrom(ctx context.Context, objs ...*storage.Policy) error {
	conn, release, err := s.acquireConn(ctx, ops.Get, "Policy")
	if err != nil {
		return err
	}
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	if err := s.copyFromPolicies(ctx, tx, objs...); err != nil {
		if err := tx.Rollback(ctx); err != nil {
			return err
		}
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (s *storeImpl) upsert(ctx context.Context, objs ...*storage.Policy) error {
	conn, release, err := s.acquireConn(ctx, ops.Get, "Policy")
	if err != nil {
		return err
	}
	defer release()

	for _, obj := range objs {
		batch := &pgx.Batch{}
		if err := insertIntoPolicies(ctx, batch, obj); err != nil {
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
func (s *storeImpl) Upsert(ctx context.Context, obj *storage.Policy) error {
	return pgutils.Retry(func() error {
		return s.upsert(ctx, obj)
	})
}

// UpsertMany saves the state of multiple objects in the storage.
func (s *storeImpl) UpsertMany(ctx context.Context, objs []*storage.Policy) error {
	return pgutils.Retry(func() error {
		// Lock since copyFrom requires a delete first before being executed.  If multiple processes are updating
		// same subset of rows, both deletes could occur before the copyFrom resulting in unique constraint
		// violations
		if len(objs) < batchAfter {
			s.mutex.RLock()
			defer s.mutex.RUnlock()

			return s.upsert(ctx, objs...)
		}
		s.mutex.Lock()
		defer s.mutex.Unlock()

		return s.copyFrom(ctx, objs...)
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
func (s *storeImpl) DeleteByQuery(ctx context.Context, query *v1.Query) error {
	var sacQueryFilter *v1.Query

	q := search.ConjunctionQuery(
		sacQueryFilter,
		query,
	)

	return pgSearch.RunDeleteRequestForSchema(ctx, schema, q, s.db)
}

// DeleteMany removes the objects associated to the specified IDs from the store.
func (s *storeImpl) DeleteMany(ctx context.Context, identifiers []string) error {
	var sacQueryFilter *v1.Query

	// Batch the deletes
	localBatchSize := deleteBatchSize
	numRecordsToDelete := len(identifiers)
	for {
		if len(identifiers) == 0 {
			break
		}

		if len(identifiers) < localBatchSize {
			localBatchSize = len(identifiers)
		}

		identifierBatch := identifiers[:localBatchSize]
		q := search.ConjunctionQuery(
			sacQueryFilter,
			search.NewQueryBuilder().AddDocIDs(identifierBatch...).ProtoQuery(),
		)

		if err := pgSearch.RunDeleteRequestForSchema(ctx, schema, q, s.db); err != nil {
			return errors.Wrapf(err, "unable to delete the records.  Successfully deleted %d out of %d", numRecordsToDelete-len(identifiers), numRecordsToDelete)
		}

		// Move the slice forward to start the next batch
		identifiers = identifiers[localBatchSize:]
	}

	return nil
}

// Count returns the number of objects in the store.
func (s *storeImpl) Count(ctx context.Context) (int, error) {
	var sacQueryFilter *v1.Query
	return pgSearch.RunCountRequestForSchema(ctx, schema, sacQueryFilter, s.db)
}

// Exists returns if the ID exists in the store.
func (s *storeImpl) Exists(ctx context.Context, id string) (bool, error) {
	var sacQueryFilter *v1.Query

	q := search.ConjunctionQuery(
		sacQueryFilter,
		search.NewQueryBuilder().AddDocIDs(id).ProtoQuery(),
	)

	count, err := pgSearch.RunCountRequestForSchema(ctx, schema, q, s.db)
	// With joins and multiple paths to the scoping resources, it can happen that the Count query for an object identifier
	// returns more than 1, despite the fact that the identifier is unique in the table.
	return count > 0, err
}

// Get returns the object, if it exists from the store.
func (s *storeImpl) Get(ctx context.Context, id string) (*storage.Policy, bool, error) {
	var sacQueryFilter *v1.Query

	q := search.ConjunctionQuery(
		sacQueryFilter,
		search.NewQueryBuilder().AddDocIDs(id).ProtoQuery(),
	)

	data, err := pgSearch.RunGetQueryForSchema[storage.Policy](ctx, schema, q, s.db)
	if err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}

	return data, true, nil
}

// GetByQuery returns the objects from the store matching the query.
func (s *storeImpl) GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.Policy, error) {
	var sacQueryFilter *v1.Query

	pagination := query.GetPagination()
	q := search.ConjunctionQuery(
		sacQueryFilter,
		query,
	)
	q.Pagination = pagination

	rows, err := pgSearch.RunGetManyQueryForSchema[storage.Policy](ctx, schema, q, s.db)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return rows, nil
}

// GetMany returns the objects specified by the IDs from the store as well as the index in the missing indices slice.
func (s *storeImpl) GetMany(ctx context.Context, identifiers []string) ([]*storage.Policy, []int, error) {
	if len(identifiers) == 0 {
		return nil, nil, nil
	}

	var sacQueryFilter *v1.Query

	q := search.ConjunctionQuery(
		sacQueryFilter,
		search.NewQueryBuilder().AddDocIDs(identifiers...).ProtoQuery(),
	)

	rows, err := pgSearch.RunGetManyQueryForSchema[storage.Policy](ctx, schema, q, s.db)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			missingIndices := make([]int, 0, len(identifiers))
			for i := range identifiers {
				missingIndices = append(missingIndices, i)
			}
			return nil, missingIndices, nil
		}
		return nil, nil, err
	}
	resultsByID := make(map[string]*storage.Policy, len(rows))
	for _, msg := range rows {
		resultsByID[msg.GetId()] = msg
	}
	missingIndices := make([]int, 0, len(identifiers)-len(resultsByID))
	// It is important that the elems are populated in the same order as the input identifiers
	// slice, since some calling code relies on that to maintain order.
	elems := make([]*storage.Policy, 0, len(resultsByID))
	for i, identifier := range identifiers {
		if result, ok := resultsByID[identifier]; !ok {
			missingIndices = append(missingIndices, i)
		} else {
			elems = append(elems, result)
		}
	}
	return elems, missingIndices, nil
}

// GetIDs returns all the IDs for the store.
func (s *storeImpl) GetIDs(ctx context.Context) ([]string, error) {
	var sacQueryFilter *v1.Query

	result, err := pgSearch.RunSearchRequestForSchema(ctx, schema, sacQueryFilter, s.db)
	if err != nil {
		return nil, err
	}

	identifiers := make([]string, 0, len(result))
	for _, entry := range result {
		identifiers = append(identifiers, entry.ID)
	}

	return identifiers, nil
}

// GetAll retrieves all objects from the store.
func (s *storeImpl) GetAll(ctx context.Context) ([]*storage.Policy, error) {
	var objs []*storage.Policy
	err := s.Walk(ctx, func(obj *storage.Policy) error {
		objs = append(objs, obj)
		return nil
	})
	return objs, err
}

// Walk iterates over all of the objects in the store and applies the closure.
func (s *storeImpl) Walk(ctx context.Context, fn func(obj *storage.Policy) error) error {
	var sacQueryFilter *v1.Query
	fetcher, closer, err := pgSearch.RunCursorQueryForSchema[storage.Policy](ctx, schema, sacQueryFilter, s.db)
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

//// Stubs for satisfying legacy interfaces

// RenamePolicyCategory is not implemented in postgres mode.
func (s *storeImpl) RenamePolicyCategory(_ *v1.RenamePolicyCategoryRequest) error {
	return errors.New("unimplemented")
}

// DeletePolicyCategory is not implemented in postgres mode.
func (s *storeImpl) DeletePolicyCategory(_ *v1.DeletePolicyCategoryRequest) error {
	return errors.New("unimplemented")
}

// AckKeysIndexed acknowledges the passed keys were indexed.
func (s *storeImpl) AckKeysIndexed(_ context.Context, _ ...string) error {
	return nil
}

// GetKeysToIndex returns the keys that need to be indexed.
func (s *storeImpl) GetKeysToIndex(_ context.Context) ([]string, error) {
	return nil, nil
}

//// Interface functions - END
