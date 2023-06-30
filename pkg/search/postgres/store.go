package postgres

import (
	"context"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
)

// PermissionChecker is a permission checker that could be used by GenericStore
type PermissionChecker interface {
	ExistsAllowed(ctx context.Context) (bool, error)
}

type primaryKeyGetter[T any, PT unmarshaler[T]] func(obj PT) string
type durationTimeSetter func(start time.Time, op ops.Op)

// GenericStore implements subset of Store interface for resources with single ID.
type GenericStore[T any, PT unmarshaler[T]] struct {
	db                               postgres.DB
	targetResource                   permissions.ResourceMetadata
	schema                           *walker.Schema
	setPostgresOperationDurationTime durationTimeSetter
	setAcquireDBConnDuration         durationTimeSetter
	permissionChecker                PermissionChecker
	pkGetter                         primaryKeyGetter[T, PT]
}

// NewGenericStore returns new subStore implementation for given resource.
// subStore implements subset of Store operations.
func NewGenericStore[T any, PT unmarshaler[T]](
	db postgres.DB,
	targetResource permissions.ResourceMetadata,
	schema *walker.Schema,
	setPostgresOperationDurationTime durationTimeSetter,
	setAcquireDBConnDuration durationTimeSetter,
	pkGetter primaryKeyGetter[T, PT],
) *GenericStore[T, PT] {
	return &GenericStore[T, PT]{
		db:                               db,
		targetResource:                   targetResource,
		schema:                           schema,
		setPostgresOperationDurationTime: setPostgresOperationDurationTime,
		setAcquireDBConnDuration:         setAcquireDBConnDuration,
		pkGetter:                         pkGetter,
	}
}

// NewGenericStoreWithPermissionChecker returns new subStore implementation for given resource.
// subStore implements subset of Store operations.
func NewGenericStoreWithPermissionChecker[T any, PT unmarshaler[T]](
	db postgres.DB,
	checker PermissionChecker,
	schema *walker.Schema,
	setPostgresOperationDurationTime durationTimeSetter,
	setAcquireDBConnDuration durationTimeSetter,
	pkGetter primaryKeyGetter[T, PT],
) *GenericStore[T, PT] {
	return &GenericStore[T, PT]{
		db:                               db,
		schema:                           schema,
		setPostgresOperationDurationTime: setPostgresOperationDurationTime,
		setAcquireDBConnDuration:         setAcquireDBConnDuration,
		permissionChecker:                checker,
		pkGetter:                         pkGetter,
	}
}

func (s *GenericStore[T, PT]) hasPermissionsChecker() bool {
	return s.permissionChecker != nil
}

// AcquireConn returns Acquires new connection from DB.
func (s *GenericStore[T, PT]) AcquireConn(ctx context.Context, op ops.Op) (*postgres.Conn, error) {
	defer s.setAcquireDBConnDuration(time.Now(), op)
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	return conn, nil
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
