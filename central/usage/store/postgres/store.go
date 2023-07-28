package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	baseTable = pkgSchema.UsageTableName

	getStmt    = "SELECT timestamp, serialized FROM " + baseTable + " WHERE timestamp >= $1 AND timestamp < $2"
	upsertStmt = "INSERT INTO " + baseTable + "(timestamp, serialized) VALUES ($1, $2)" +
		" ON CONFLICT(timestamp) UPDATE SET serialized=EXCLUDED.serialized"

	operation = "Usage"
)

var (
	log            = logging.LoggerForModule()
	schema         = pkgSchema.UsageSchema
	targetResource = resources.Administration
	zeroTime       = time.Unix(0, 0).UTC()
)

// Store is the interface to interact with the storage for storage.Usage.
//
//go:generate mockgen-wrapper
type Store interface {
	Get(ctx context.Context, from *types.Timestamp, to *types.Timestamp) ([]*storage.Usage, error)
	Upsert(ctx context.Context, rec *storage.Usage) error
}

type storeImpl struct {
	db    postgres.DB
	mutex sync.Mutex
}

// New returns a new Store instance using the provided sql instance.
func New(db postgres.DB) Store {
	return &storeImpl{
		db: db,
	}
}

func checkScope(ctx context.Context, am storage.Access) error {
	if !sac.GlobalAccessScopeChecker(ctx).AccessMode(am).
		Resource(targetResource).IsAllowed() {
		return sac.ErrResourceAccessDenied
	}
	return nil
}

// Upsert saves the current state of an object in storage.
func (s *storeImpl) Upsert(ctx context.Context, obj *storage.Usage) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Add, operation)

	if err := checkScope(ctx, storage.Access_READ_WRITE_ACCESS); err != nil {
		return err
	}

	err := pgutils.Retry(func() error {
		return s.retryableInsert(ctx, obj)
	})
	return errors.Wrap(err, "cannot insert metrics")
}

func (s *storeImpl) retryableInsert(ctx context.Context, rec *storage.Usage) error {
	serialized, err := rec.Marshal()
	if err != nil {
		return err
	}

	conn, release, err := s.acquireConn(ctx, ops.Get, operation)
	if err != nil {
		return err
	}
	defer release()
	_, err = conn.Exec(ctx, upsertStmt, protoconv.ConvertTimestampToTimeOrNow(rec.GetTimestamp()), serialized)
	return err
}

// Get returns the object, if it exists from the store.
func (s *storeImpl) Get(ctx context.Context, from *types.Timestamp, to *types.Timestamp) ([]*storage.Usage, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, operation)

	if err := checkScope(ctx, storage.Access_READ_ACCESS); err != nil {
		return nil, nil
	}

	f := protoconv.ConvertTimestampToTimeOrDefault(from, zeroTime)
	t := protoconv.ConvertTimestampToTimeOrNow(to)

	r, err := pgutils.Retry2(func() ([]*storage.Usage, error) {
		return s.retryableGet(ctx, &f, &t)
	})
	return r, errors.Wrap(err, "cannot get metrics from db")
}

func (s *storeImpl) retryableGet(ctx context.Context, from *time.Time, to *time.Time) ([]*storage.Usage, error) {
	conn, release, err := s.acquireConn(ctx, ops.Get, operation)
	if err != nil {
		return nil, err
	}
	defer release()

	rows, err := conn.Query(ctx, getStmt, from, to)
	if err != nil {
		return nil, pgutils.ErrNilIfNoRows(err)
	}

	result := []*storage.Usage{}
	for rows.Next() {
		var ts *time.Time
		var obj []byte
		if err := rows.Scan(&ts, &obj); err != nil {
			return nil, pgutils.ErrNilIfNoRows(err)
		}
		var value *storage.Usage
		if err := value.Unmarshal(obj); err != nil {
			return nil, err
		}
		result = append(result, value)
	}

	return result, nil
}

func (s *storeImpl) acquireConn(ctx context.Context, op ops.Op, typ string) (*postgres.Conn, func(), error) {
	defer metrics.SetAcquireDBConnDuration(time.Now(), op, typ)
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot acquire connection for %s %s: %w", op.String(), typ, err)
	}
	return conn, conn.Release, nil
}
