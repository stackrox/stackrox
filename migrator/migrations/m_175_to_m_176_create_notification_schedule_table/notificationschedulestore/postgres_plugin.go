package postgres

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
)

const (
	getStmt    = "SELECT serialized FROM notification_schedules LIMIT 1"
	deleteStmt = "DELETE FROM notification_schedules"
)

// Store is the interface to interact with the storage for storage.NotificationSchedule
type Store interface {
	Get(ctx context.Context) (*storage.NotificationSchedule, bool, error)
	Upsert(ctx context.Context, obj *storage.NotificationSchedule) error
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

func insertIntoNotificationSchedules(ctx context.Context, tx *postgres.Tx, obj *storage.NotificationSchedule) error {
	serialized, marshalErr := obj.Marshal()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		// parent primary keys start
		serialized,
	}

	finalStr := "INSERT INTO notification_schedules (serialized) VALUES($1)"
	_, err := tx.Exec(ctx, finalStr, values...)
	if err != nil {
		return err
	}
	return nil
}

// Upsert saves the current state of an object in storage.
func (s *storeImpl) Upsert(ctx context.Context, obj *storage.NotificationSchedule) error {
	return pgutils.Retry(func() error {
		return s.retryableUpsert(ctx, obj)
	})
}

func (s *storeImpl) retryableUpsert(ctx context.Context, obj *storage.NotificationSchedule) error {
	conn, release, err := s.acquireConn(ctx, ops.Get, "NotificationSchedule")
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

	if err := insertIntoNotificationSchedules(ctx, tx, obj); err != nil {
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

// Get returns the object, if it exists from the store.
func (s *storeImpl) Get(ctx context.Context) (*storage.NotificationSchedule, bool, error) {
	return pgutils.Retry3(func() (*storage.NotificationSchedule, bool, error) {
		return s.retryableGet(ctx)
	})
}

func (s *storeImpl) retryableGet(ctx context.Context) (*storage.NotificationSchedule, bool, error) {
	conn, release, err := s.acquireConn(ctx, ops.Get, "NotificationSchedule")
	if err != nil {
		return nil, false, err
	}
	defer release()

	row := conn.QueryRow(ctx, getStmt)
	var data []byte
	if err := row.Scan(&data); err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}

	var msg storage.NotificationSchedule
	if err := msg.Unmarshal(data); err != nil {
		return nil, false, err
	}
	return &msg, true, nil
}

func (s *storeImpl) acquireConn(ctx context.Context, _ ops.Op, _ string) (*postgres.Conn, func(), error) {
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, nil, err
	}
	return conn, conn.Release, nil
}

// Delete removes the singleton from the store
func (s *storeImpl) Delete(ctx context.Context) error {
	return pgutils.Retry(func() error {
		return s.retryableDelete(ctx)
	})
}

func (s *storeImpl) retryableDelete(ctx context.Context) error {
	conn, release, err := s.acquireConn(ctx, ops.Remove, "NotificationSchedule")
	if err != nil {
		return err
	}
	defer release()

	if _, err := conn.Exec(ctx, deleteStmt); err != nil {
		return err
	}
	return nil
}
