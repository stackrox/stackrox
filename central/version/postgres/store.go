package postgres

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	getStmt    = "SELECT serialized FROM versions LIMIT 1"
	deleteStmt = "DELETE FROM versions"
)

var (
	log    = logging.LoggerForModule()
	schema = pkgSchema.VersionsSchema
)

// Store access versions in database
type Store interface {
	Get(ctx context.Context) (*storage.Version, bool, error)
	Upsert(ctx context.Context, obj *storage.Version) error
	Delete(ctx context.Context) error
}

type storeImpl struct {
	db    *postgres.DB
	mutex sync.Mutex
}

// New returns a new Store instance using the provided sql instance.
func New(db *postgres.DB) Store {
	return &storeImpl{
		db: db,
	}
}

func insertIntoVersions(ctx context.Context, tx pgx.Tx, obj *storage.Version) error {
	serialized, marshalErr := obj.Marshal()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		// parent primary keys start
		serialized,
	}

	finalStr := "INSERT INTO versions (serialized) VALUES($1)"
	_, err := tx.Exec(ctx, finalStr, values...)
	if err != nil {
		return err
	}
	return nil
}

func (s *storeImpl) Upsert(ctx context.Context, obj *storage.Version) error {
	return pgutils.Retry(func() error {
		return s.retryableUpsert(ctx, obj)
	})
}

func (s *storeImpl) retryableUpsert(ctx context.Context, obj *storage.Version) error {
	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_WRITE_ACCESS)
	if !scopeChecker.IsAllowed() {
		return sac.ErrResourceAccessDenied
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

	if err := insertIntoVersions(ctx, tx, obj); err != nil {
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

// Get returns the object, if it exists from the store
func (s *storeImpl) Get(ctx context.Context) (*storage.Version, bool, error) {
	return pgutils.Retry3(func() (*storage.Version, bool, error) {
		return s.retryableGet(ctx)
	})
}

func (s *storeImpl) retryableGet(ctx context.Context) (*storage.Version, bool, error) {
	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_ACCESS)
	if !scopeChecker.IsAllowed() {
		return nil, false, nil
	}

	conn, release, err := s.acquireConn(ctx, ops.Get, "Version")
	if err != nil {
		return nil, false, err
	}
	defer release()

	row := conn.QueryRow(ctx, getStmt)
	var data []byte
	if err := row.Scan(&data); err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}

	var msg storage.Version
	if err := msg.Unmarshal(data); err != nil {
		return nil, false, err
	}
	return &msg, true, nil
}

func (s *storeImpl) acquireConn(ctx context.Context, op ops.Op, typ string) (*postgres.Conn, func(), error) {
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, nil, err
	}
	return conn, conn.Release, nil
}

// Delete removes the specified ID from the store
func (s *storeImpl) Delete(ctx context.Context) error {
	return pgutils.Retry(func() error {
		return s.retryableDelete(ctx)
	})
}

func (s *storeImpl) retryableDelete(ctx context.Context) error {
	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_WRITE_ACCESS)
	if !scopeChecker.IsAllowed() {
		return sac.ErrResourceAccessDenied
	}

	conn, release, err := s.acquireConn(ctx, ops.Remove, "Version")
	if err != nil {
		return err
	}
	defer release()

	if _, err := conn.Exec(ctx, deleteStmt); err != nil {
		return err
	}
	return nil
}

// Used for Testing

// Destroy is Used for Testing
func Destroy(ctx context.Context, db *postgres.DB) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS versions CASCADE")
}
