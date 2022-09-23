package store

import (
	"context"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protoconv"
	"gorm.io/gorm"
)

// The PersistentLogs store is custom as it utilizes insert only techniques and is relatively simple
const (
	persistentLogsTable = pkgSchema.PersistentLogsTableName

	countStmt = "SELECT COUNT(*) FROM persistent_logs"

	getStmt = `SELECT log, timestamp 
	FROM persistent_logs 
	WHERE timestamp between $1 and $2`
	deleteStmt = "DELETE FROM persistent_logs WHERE timestamp < $1"
	walkStmt   = "SELECT log, timestamp FROM persistent_logs"
)

var (
	log = logging.LoggerForModule()
)

// Store stores all the persistent logs.
type Store interface {
	Count(ctx context.Context) (int, error)
	Get(ctx context.Context, startTime, endTime *types.Timestamp) ([]*storage.PersistentLog, bool, error)
	Upsert(ctx context.Context, obj *storage.PersistentLog) error
	DeleteBefore(ctx context.Context, timestamp *types.Timestamp) error
	Walk(ctx context.Context, fn func(obj *storage.PersistentLog) error) error

	GetAllPersistentLogs(ctx context.Context) ([]*storage.PersistentLog, error)
}

type persistentLogStoreImpl struct {
	db *pgxpool.Pool
}

func (s *persistentLogStoreImpl) insertIntoPersistentLog(ctx context.Context, tx pgx.Tx, obj *storage.PersistentLog) error {

	values := []interface{}{
		// parent primary keys start
		obj.GetLog(),
		pgutils.NilOrTime(obj.GetTimestamp()),
	}

	finalStr := "INSERT INTO persistent_logs (log, timestamp) VALUES($1, $2)"
	_, err := tx.Exec(ctx, finalStr, values...)
	if err != nil {
		return err
	}

	return nil
}

// New returns a new Store instance using the provided sql instance.
func New(db *pgxpool.Pool) Store {
	return &persistentLogStoreImpl{
		db: db,
	}
}

func (s *persistentLogStoreImpl) upsert(ctx context.Context, objs ...*storage.PersistentLog) error {
	conn, release, err := s.acquireConn(ctx, ops.Get, "PersistentLogs")
	if err != nil {
		return err
	}
	defer release()

	// Moved the transaction outside the loop which greatly improved the performance of these individual inserts.
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	for _, obj := range objs {

		if err := s.insertIntoPersistentLog(ctx, tx, obj); err != nil {
			if err := tx.Rollback(ctx); err != nil {
				return err
			}
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (s *persistentLogStoreImpl) Upsert(ctx context.Context, obj *storage.PersistentLog) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Upsert, "PersistentLogs")

	return s.upsert(ctx, obj)
}

// Count returns the number of objects in the store
func (s *persistentLogStoreImpl) Count(ctx context.Context) (int, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Count, "PersistentLogs")

	row := s.db.QueryRow(ctx, countStmt)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// Get returns the object, if it exists from the store
func (s *persistentLogStoreImpl) Get(ctx context.Context, startTime, endTime *types.Timestamp) ([]*storage.PersistentLog, bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "NetworkFlow")

	conn, release, err := s.acquireConn(ctx, ops.Get, "NetworkFlow")
	if err != nil {
		return nil, false, err
	}
	defer release()

	// We can discuss this a bit, but this statement should only ever return 1 row.  Doing it this way allows
	// us to use the readRows function
	rows, err := conn.Query(ctx, getStmt, pgutils.NilOrTime(startTime), pgutils.NilOrTime(endTime))
	if err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}
	defer rows.Close()

	logs, err := s.readRows(rows)
	if err != nil || logs == nil {
		return nil, false, err
	}

	return logs, true, nil
}

func (s *persistentLogStoreImpl) acquireConn(ctx context.Context, op ops.Op, typ string) (*pgxpool.Conn, func(), error) {
	defer metrics.SetAcquireDBConnDuration(time.Now(), op, typ)
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, nil, err
	}
	return conn, conn.Release, nil
}

func (s *persistentLogStoreImpl) readRows(rows pgx.Rows) ([]*storage.PersistentLog, error) {
	var logs []*storage.PersistentLog

	for rows.Next() {
		var logText string
		var timestamp *time.Time

		if err := rows.Scan(&logText, &timestamp); err != nil {
			return nil, pgutils.ErrNilIfNoRows(err)
		}

		var ts *types.Timestamp
		if timestamp != nil {
			ts = protoconv.MustConvertTimeToTimestamp(*timestamp)
		}

		logEntry := &storage.PersistentLog{
			Log:       logText,
			Timestamp: ts,
		}

		logs = append(logs, logEntry)
	}

	log.Debugf("Read returned %d flows", len(logs))
	return logs, nil
}

// DeleteBefore removes the specified ID from the store
func (s *persistentLogStoreImpl) DeleteBefore(ctx context.Context, timestamp *types.Timestamp) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Remove, "NetworkFlow")

	conn, release, err := s.acquireConn(ctx, ops.Remove, "NetworkFlow")
	if err != nil {
		return err
	}
	defer release()

	if _, err := conn.Exec(ctx, deleteStmt, pgutils.NilOrTime(timestamp)); err != nil {
		return err
	}
	return nil
}

// Walk iterates over all of the objects in the store and applies the closure
func (s *persistentLogStoreImpl) Walk(ctx context.Context, fn func(obj *storage.PersistentLog) error) error {
	rows, err := s.db.Query(ctx, walkStmt)
	if err != nil {
		return pgutils.ErrNilIfNoRows(err)
	}
	defer rows.Close()
	for rows.Next() {
		var logText string
		var timestamp *time.Time

		if err := rows.Scan(&logText, &timestamp); err != nil {
			return err
		}

		var ts *types.Timestamp
		if timestamp != nil {
			ts = protoconv.MustConvertTimeToTimestamp(*timestamp)
		}

		logEntry := &storage.PersistentLog{
			Log:       logText,
			Timestamp: ts,
		}

		if err := fn(logEntry); err != nil {
			return err
		}
	}
	return nil
}

// GetAllPersistentLogs returns the object, if it exists from the store, timestamp and error
func (s *persistentLogStoreImpl) GetAllPersistentLogs(ctx context.Context) ([]*storage.PersistentLog, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "NetworkFlow")

	var rows pgx.Rows
	var err error

	rows, err = s.db.Query(ctx, walkStmt)
	if err != nil {
		return nil, pgutils.ErrNilIfNoRows(err)
	}
	defer rows.Close()

	flows, err := s.readRows(rows)
	if err != nil {
		return nil, pgutils.ErrNilIfNoRows(err)
	}

	return flows, nil
}

//// Used for testing

func dropTablePersistentLog(ctx context.Context, db *pgxpool.Pool) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS persistent_logs CASCADE")
}

// Destroy destroys the tables
func Destroy(ctx context.Context, db *pgxpool.Pool) {
	dropTablePersistentLog(ctx, db)
}

// CreateTableAndNewStore returns a new Store instance for testing
func CreateTableAndNewStore(ctx context.Context, db *pgxpool.Pool, gormDB *gorm.DB) Store {
	pkgSchema.ApplySchemaForTable(ctx, gormDB, persistentLogsTable)
	return New(db)
}

//// Stubs for satisfying legacy interfaces

// AckKeysIndexed acknowledges the passed keys were indexed
func (s *persistentLogStoreImpl) AckKeysIndexed(ctx context.Context, keys ...string) error {
	return nil
}

// GetKeysToIndex returns the keys that need to be indexed
func (s *persistentLogStoreImpl) GetKeysToIndex(ctx context.Context) ([]string, error) {
	return nil, nil
}
