package postgres

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
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

// GenericSingleIDStore implements subset of Store interface for resources with single ID.
type GenericSingleIDStore[T any, PT singleID[T]] struct {
	db                               postgres.DB
	typ                              string
	targetResource                   permissions.ResourceMetadata
	schema                           *walker.Schema
	setPostgresOperationDurationTime func(start time.Time, op ops.Op, t string)
	permissionChecker                PermissionChecker
}

type singleID[T any] interface {
	proto.Unmarshaler
	*T
}

// NewGenericSingleIDStore returns new subStore implementation for given resource.
// subStore implements subset of Store operations.
func NewGenericSingleIDStore[T any, PT singleID[T]](db postgres.DB, typ string, targetResource permissions.ResourceMetadata, schema *walker.Schema, setPostgresOperationDurationTime func(start time.Time, op ops.Op, t string)) *GenericSingleIDStore[T, PT] {
	return &GenericSingleIDStore[T, PT]{
		db:                               db,
		typ:                              typ,
		targetResource:                   targetResource,
		schema:                           schema,
		setPostgresOperationDurationTime: setPostgresOperationDurationTime,
	}
}

// NewGenericSingleIDStoreWithPermissionChecker returns new subStore implementation for given resource.
// subStore implements subset of Store operations.
func NewGenericSingleIDStoreWithPermissionChecker[T any, PT singleID[T]](db postgres.DB, typ string, checker PermissionChecker, schema *walker.Schema, setPostgresOperationDurationTime func(start time.Time, op ops.Op, t string)) *GenericSingleIDStore[T, PT] {
	return &GenericSingleIDStore[T, PT]{
		db:                               db,
		typ:                              typ,
		schema:                           schema,
		setPostgresOperationDurationTime: setPostgresOperationDurationTime,
		permissionChecker:                checker,
	}
}

func (s *GenericSingleIDStore[T, PT]) hasPermissionsChecker() bool {
	return s.permissionChecker != nil
}

// Count returns the number of objects in the store.
func (s *GenericSingleIDStore[T, PT]) Count(ctx context.Context) (int, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Count, s.typ)

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

// DeleteByQuery removes the objects from the store based on the passed query.
func (s *GenericSingleIDStore[T, PT]) DeleteByQuery(ctx context.Context, query *v1.Query) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Remove, s.typ)

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
func (s *GenericSingleIDStore[T, PT]) Delete(ctx context.Context, id string) error {
	q := search.NewQueryBuilder().AddDocIDs(id).ProtoQuery()
	return s.DeleteByQuery(ctx, q)
}

// DeleteMany removes the objects associated to the specified IDs from the store.
func (s *GenericSingleIDStore[T, PT]) DeleteMany(ctx context.Context, identifiers []string) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.RemoveMany, s.typ)

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

// Exists returns if the ID exists in the store.
func (s *GenericSingleIDStore[T, PT]) Exists(ctx context.Context, id string) (bool, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Exists, s.typ)

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
func (s *GenericSingleIDStore[T, PT]) Get(ctx context.Context, id string) (PT, bool, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Get, s.typ)

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

// GetByQuery returns the objects from the store matching the query.
func (s *GenericSingleIDStore[T, PT]) GetByQuery(ctx context.Context, query *v1.Query) ([]*T, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.GetByQuery, s.typ)

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
func (s *GenericSingleIDStore[T, PT]) GetIDs(ctx context.Context) ([]string, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.GetAll, s.typ+"IDs")
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

// GetAll retrieves all objects from the store.
func (s *GenericSingleIDStore[T, PT]) GetAll(ctx context.Context) ([]*T, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.GetAll, "Notifier")

	var objs []*T
	err := s.Walk(ctx, func(obj PT) error {
		objs = append(objs, (*T)(obj))
		return nil
	})
	return objs, err
}

// Walk iterates over all the objects in the store and applies the closure.
func (s *GenericSingleIDStore[T, PT]) Walk(ctx context.Context, fn func(obj PT) error) error {
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
