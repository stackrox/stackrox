package postgres

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

const (
	tableName  = pkgSchema.SystemInfosTableName
	getStmt    = "SELECT serialized FROM " + tableName + " LIMIT 1"
	insertStmt = "INSERT INTO " + tableName + " (serialized) VALUES($1)"
	deleteStmt = "DELETE FROM " + tableName
)

var (
	sysInfoSAC = sac.ForResource(resources.Administration)
)

// Store provides functionality to read and write system info.
type Store interface {
	Get(ctx context.Context) (*storage.SystemInfo, bool, error)
	Upsert(ctx context.Context, obj *storage.SystemInfo) error
	Delete(ctx context.Context) error
}

type storeImpl struct {
	db postgres.DB
}

// New returns a new Store instance using the provided sql instance.
func New(db postgres.DB) Store {
	return &storeImpl{
		db: db,
	}
}

func (s *storeImpl) Get(ctx context.Context) (*storage.SystemInfo, bool, error) {
	return pgutils.Retry3(func() (*storage.SystemInfo, bool, error) {
		return s.get(ctx)
	})
}

func (s *storeImpl) Upsert(ctx context.Context, obj *storage.SystemInfo) error {
	return pgutils.Retry(func() error {
		return s.retryableUpsert(ctx, obj)
	})
}

func (s *storeImpl) Delete(ctx context.Context) error {
	return pgutils.Retry(func() error {
		return s.retryableDelete(ctx)
	})
}

func (s *storeImpl) get(ctx context.Context) (*storage.SystemInfo, bool, error) {
	if ok, err := sysInfoSAC.ReadAllowed(ctx); err != nil {
		return nil, false, err
	} else if !ok {
		return nil, false, errox.NotAuthorized
	}

	conn, release, err := s.acquireConn(ctx, ops.Get, "SystemInfo")
	if err != nil {
		return nil, false, err
	}
	defer release()

	row := conn.QueryRow(ctx, getStmt)
	var data []byte
	if err := row.Scan(&data); err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}

	var msg storage.SystemInfo
	if err := msg.Unmarshal(data); err != nil {
		return nil, false, err
	}
	return &msg, true, nil
}

func (s *storeImpl) retryableUpsert(ctx context.Context, obj *storage.SystemInfo) error {
	if ok, err := sysInfoSAC.WriteAllowed(ctx); err != nil || !ok {
		return err
	}

	conn, release, err := s.acquireConn(ctx, ops.Get, "Version")
	if err != nil {
		return err
	}
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, deleteStmt); err != nil {
		return err
	}

	if err := insert(ctx, tx, obj); err != nil {
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

func (s *storeImpl) retryableDelete(ctx context.Context) error {
	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_WRITE_ACCESS)
	if !scopeChecker.IsAllowed() {
		return sac.ErrResourceAccessDenied
	}

	conn, release, err := s.acquireConn(ctx, ops.Remove, "SystemInfo")
	if err != nil {
		return err
	}
	defer release()

	if _, err := conn.Exec(ctx, deleteStmt); err != nil {
		return err
	}
	return nil
}

func (s *storeImpl) acquireConn(ctx context.Context, op ops.Op, typ string) (*postgres.Conn, func(), error) {
	defer metrics.SetAcquireDBConnDuration(time.Now(), op, typ)
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, nil, err
	}
	return conn, conn.Release, nil
}

func insert(ctx context.Context, tx *postgres.Tx, obj *storage.SystemInfo) error {
	serialized, marshalErr := obj.Marshal()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		serialized,
	}

	_, err := tx.Exec(ctx, insertStmt, values...)
	if err != nil {
		return err
	}
	return nil
}

// Used for Testing

// Destroy is Used for Testing
func Destroy(ctx context.Context, db postgres.DB) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS "+tableName+" CASCADE")
}
