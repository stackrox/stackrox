package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
)

const (
	cursorBatchSize = 50
	deleteBatchSize = 5000
)

type PermissionChecker interface {
	CountAllowed(ctx context.Context) (bool, error)
	ExistsAllowed(ctx context.Context) (bool, error)
	GetAllowed(ctx context.Context) (bool, error)
	WalkAllowed(ctx context.Context) (bool, error)
	DeleteAllowed(ctx context.Context, keys ...sac.ScopeKey) (bool, error)
	DeleteManyAllowed(ctx context.Context, keys ...sac.ScopeKey) (bool, error)
}

type primaryKeyGetter[T any, PT unmarshaler[T]] func(obj PT) string
type durationTimeSetter func(start time.Time, op ops.Op)

// GenericStore implements subset of Store interface for resources with single ID.
type GenericStore[T any, PT unmarshaler[T]] struct {
	db                               postgres.DB
	targetResource                   permissions.ResourceMetadata
	schema                           *walker.Schema
	setPostgresOperationDurationTime durationTimeSetter
	permissionChecker                PermissionChecker
	pkGetter                         primaryKeyGetter[T, PT]
}

// GenericSingleIDStore implements subset of Store interface for resources with single ID.
type GenericSingleIDStore[T any, PT unmarshaler[T]] struct {
	*GenericStore[T, PT]
}

// NewGenericStore returns new subStore implementation for given resource.
// subStore implements subset of Store operations.
func NewGenericStore[T any, PT unmarshaler[T]](db postgres.DB, targetResource permissions.ResourceMetadata, schema *walker.Schema, setPostgresOperationDurationTime durationTimeSetter, pkGetter primaryKeyGetter[T, PT]) *GenericStore[T, PT] {
	return &GenericStore[T, PT]{
		db:                               db,
		targetResource:                   targetResource,
		schema:                           schema,
		setPostgresOperationDurationTime: setPostgresOperationDurationTime,
		pkGetter:                         pkGetter,
	}
}

// NewGenericSingleIDStore returns new subStore implementation for given resource.
// subStore implements subset of Store operations.
func NewGenericSingleIDStore[T any, PT unmarshaler[T]](db postgres.DB, targetResource permissions.ResourceMetadata, schema *walker.Schema, setPostgresOperationDurationTime durationTimeSetter, pkGetter primaryKeyGetter[T, PT]) *GenericSingleIDStore[T, PT] {
	return &GenericSingleIDStore[T, PT]{GenericStore: &GenericStore[T, PT]{
		db:                               db,
		targetResource:                   targetResource,
		schema:                           schema,
		setPostgresOperationDurationTime: setPostgresOperationDurationTime,
		pkGetter:                         pkGetter,
	},
	}
}

// NewGenericSingleIDStoreWithPermissionChecker returns new subStore implementation for given resource.
// subStore implements subset of Store operations.
func NewGenericSingleIDStoreWithPermissionChecker[T any, PT unmarshaler[T]](db postgres.DB, checker PermissionChecker, schema *walker.Schema, setPostgresOperationDurationTime durationTimeSetter, pkGetter primaryKeyGetter[T, PT]) *GenericSingleIDStore[T, PT] {
	return &GenericSingleIDStore[T, PT]{GenericStore: &GenericStore[T, PT]{
		db:                               db,
		schema:                           schema,
		setPostgresOperationDurationTime: setPostgresOperationDurationTime,
		permissionChecker:                checker,
		pkGetter:                         pkGetter,
	},
	}
}

func (s *GenericStore[T, PT]) hasPermissionsChecker() bool {
	return s.permissionChecker != nil
}

// Count returns the number of objects in the store.
func (s *GenericStore[T, PT]) Count(ctx context.Context) (int, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Count)

	var sacQueryFilter *v1.Query
	if s.hasPermissionsChecker() {
		if ok, err := s.permissionChecker.CountAllowed(ctx); err != nil {
			return 0, err
		} else if !ok {
			return 0, sac.ErrResourceAccessDenied
		}
	} else {
		filter, err := GetReadSACQuery(ctx, s.targetResource)
		if err != nil {
			return 0, err
		}
		sacQueryFilter = filter
	}

	return RunCountRequestForSchema(ctx, s.schema, sacQueryFilter, s.db)
}

// GetByQuery returns the objects from the store matching the query.
func (s *GenericStore[T, PT]) GetByQuery(ctx context.Context, query *v1.Query) ([]*T, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.GetByQuery)

	var sacQueryFilter *v1.Query
	if s.hasPermissionsChecker() {
		if ok, err := s.permissionChecker.GetAllowed(ctx); err != nil {
			return nil, err
		} else if !ok {
			return nil, sac.ErrResourceAccessDenied
		}
	} else {
		filter, err := GetReadSACQuery(ctx, s.targetResource)
		if err != nil {
			return nil, err
		}
		sacQueryFilter = filter
	}

	pagination := query.GetPagination()
	q := search.ConjunctionQuery(
		sacQueryFilter,
		query,
	)
	q.Pagination = pagination

	rows, err := RunGetManyQueryForSchema[T, PT](ctx, s.schema, q, s.db)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return rows, nil
}

// GetIDs returns all the IDs for the store.
func (s *GenericStore[T, PT]) GetIDs(ctx context.Context) ([]string, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.GetAll)
	var sacQueryFilter *v1.Query
	if s.hasPermissionsChecker() {
		if ok, err := s.permissionChecker.GetAllowed(ctx); err != nil {
			return nil, err
		} else if !ok {
			return nil, sac.ErrResourceAccessDenied
		}
	} else {
		filter, err := GetReadSACQuery(ctx, s.targetResource)
		if err != nil {
			return nil, err
		}
		sacQueryFilter = filter
	}
	result, err := RunSearchRequestForSchema(ctx, s.schema, sacQueryFilter, s.db)
	if err != nil {
		return nil, err
	}

	identifiers := make([]string, 0, len(result))
	for _, entry := range result {
		identifiers = append(identifiers, entry.ID)
	}

	return identifiers, nil
}

// GetMany returns the objects specified by the IDs from the store as well as the index in the missing indices slice.
func (s *GenericStore[T, PT]) GetMany(ctx context.Context, identifiers []string) ([]*T, []int, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.GetMany)

	if len(identifiers) == 0 {
		return nil, nil, nil
	}

	var sacQueryFilter *v1.Query
	if s.hasPermissionsChecker() {
		if ok, err := s.permissionChecker.GetAllowed(ctx); err != nil {
			return nil, nil, err
		} else if !ok {
			return nil, nil, sac.ErrResourceAccessDenied
		}
	} else {
		filter, err := GetReadSACQuery(ctx, s.targetResource)
		if err != nil {
			return nil, nil, err
		}
		sacQueryFilter = filter
	}
	q := search.ConjunctionQuery(
		sacQueryFilter,
		search.NewQueryBuilder().AddDocIDs(identifiers...).ProtoQuery(),
	)

	rows, err := RunGetManyQueryForSchema[T, PT](ctx, s.schema, q, s.db)
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
	resultsByID := make(map[string]*T, len(rows))
	for _, msg := range rows {
		resultsByID[s.pkGetter(msg)] = msg
	}
	missingIndices := make([]int, 0, len(identifiers)-len(resultsByID))
	// It is important that the elems are populated in the same order as the input identifiers
	// slice, since some calling code relies on that to maintain order.
	elems := make([]*T, 0, len(resultsByID))
	for i, identifier := range identifiers {
		if result, ok := resultsByID[identifier]; !ok {
			missingIndices = append(missingIndices, i)
		} else {
			elems = append(elems, result)
		}
	}
	return elems, missingIndices, nil
}

// DeleteMany removes the objects associated to the specified IDs from the store.
func (s *GenericStore[T, PT]) DeleteMany(ctx context.Context, identifiers []string) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.RemoveMany)

	var sacQueryFilter *v1.Query
	if s.hasPermissionsChecker() {
		if ok, err := s.permissionChecker.DeleteManyAllowed(ctx); err != nil {
			return err
		} else if !ok {
			return sac.ErrResourceAccessDenied
		}
	} else {
		filter, err := GetReadWriteSACQuery(ctx, s.targetResource)
		if err != nil {
			return err
		}
		sacQueryFilter = filter
	}

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

		if err := RunDeleteRequestForSchema(ctx, s.schema, q, s.db); err != nil {
			return errors.Wrapf(err, "unable to delete the records.  Successfully deleted %d out of %d", numRecordsToDelete-len(identifiers), numRecordsToDelete)
		}

		// Move the slice forward to start the next batch
		identifiers = identifiers[localBatchSize:]
	}

	return nil
}

// GetAll retrieves all objects from the store.
func (s *GenericStore[T, PT]) GetAll(ctx context.Context) ([]*T, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.GetAll)

	var objs []*T
	err := s.Walk(ctx, func(obj PT) error {
		objs = append(objs, (*T)(obj))
		return nil
	})
	return objs, err
}

// Walk iterates over all the objects in the store and applies the closure.
func (s *GenericStore[T, PT]) Walk(ctx context.Context, fn func(obj PT) error) error {
	var sacQueryFilter *v1.Query
	if s.hasPermissionsChecker() {
		if ok, err := s.permissionChecker.WalkAllowed(ctx); err != nil {
			return err
		} else if !ok {
			return sac.ErrResourceAccessDenied
		}
	} else {
		filter, err := GetReadSACQuery(ctx, s.targetResource)
		if err != nil {
			return err
		}
		sacQueryFilter = filter
	}
	fetcher, closer, err := RunCursorQueryForSchema[T, PT](ctx, s.schema, sacQueryFilter, s.db)
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

// DeleteByQuery removes the objects from the store based on the passed query.
func (s *GenericStore[T, PT]) DeleteByQuery(ctx context.Context, query *v1.Query) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Remove)

	var sacQueryFilter *v1.Query
	if s.hasPermissionsChecker() {
		if ok, err := s.permissionChecker.DeleteAllowed(ctx); err != nil {
			return err
		} else if !ok {
			return sac.ErrResourceAccessDenied
		}
	} else {
		filter, err := GetReadWriteSACQuery(ctx, s.targetResource)
		if err != nil {
			return err
		}
		sacQueryFilter = filter
	}

	q := search.ConjunctionQuery(
		sacQueryFilter,
		query,
	)

	return RunDeleteRequestForSchema(ctx, s.schema, q, s.db)
}

// Delete removes the object associated to the specified ID from the store.
func (s *GenericStore[T, PT]) Delete(ctx context.Context, id string) error {
	q := search.NewQueryBuilder().AddDocIDs(id).ProtoQuery()
	return s.DeleteByQuery(ctx, q)
}

// Exists returns if the ID exists in the store.
func (s *GenericStore[T, PT]) Exists(ctx context.Context, id string) (bool, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Exists)

	var sacQueryFilter *v1.Query
	if s.hasPermissionsChecker() {
		if ok, err := s.permissionChecker.ExistsAllowed(ctx); err != nil {
			return false, err
		} else if !ok {
			return false, sac.ErrResourceAccessDenied
		}
	} else {
		filter, err := GetReadSACQuery(ctx, s.targetResource)
		if err != nil {
			return false, err
		}
		sacQueryFilter = filter
	}

	q := search.ConjunctionQuery(
		sacQueryFilter,
		search.NewQueryBuilder().AddDocIDs(id).ProtoQuery(),
	)

	count, err := RunCountRequestForSchema(ctx, s.schema, q, s.db)
	// With joins and multiple paths to the scoping resources, it can happen that the Count query for an object identifier
	// returns more than 1, despite the fact that the identifier is unique in the table.
	return count > 0, err
}

// Get returns the object, if it exists from the store.
func (s *GenericStore[T, PT]) Get(ctx context.Context, id string) (PT, bool, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Get)

	var sacQueryFilter *v1.Query
	if s.hasPermissionsChecker() {
		if ok, err := s.permissionChecker.GetAllowed(ctx); err != nil {
			return nil, false, err
		} else if !ok {
			return nil, false, sac.ErrResourceAccessDenied
		}
	} else {
		filter, err := GetReadSACQuery(ctx, s.targetResource)
		if err != nil {
			return nil, false, err
		}
		sacQueryFilter = filter
	}

	q := search.ConjunctionQuery(
		sacQueryFilter,
		search.NewQueryBuilder().AddDocIDs(id).ProtoQuery(),
	)

	data, err := RunGetQueryForSchema[T, PT](ctx, s.schema, q, s.db)
	if err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}

	return data, true, nil
}
