package postgres

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
)

const (
	getStmt    = "SELECT seqnum, version, minseqnum, lastpersisted FROM versions LIMIT 1"
	deleteStmt = "DELETE FROM versions"
)

// Store access versions in database
type Store interface {
	Get(ctx context.Context) (*storage.Version, bool, error)
	Upsert(ctx context.Context, obj *storage.Version) error
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

func insertIntoVersions(ctx context.Context, tx *postgres.Tx, obj *storage.Version) error {
	values := []interface{}{
		obj.GetSeqNum(),
		obj.GetVersion(),
		obj.GetMinSeqNum(),
		protocompat.NilOrTime(obj.GetLastPersisted()),
	}

	finalStr := "INSERT INTO versions (seqnum, version, minseqnum, lastpersisted) VALUES($1, $2, $3, $4)"
	_, err := tx.Exec(ctx, finalStr, values...)
	if err != nil {
		return err
	}
	return nil
}

func (s *storeImpl) Upsert(ctx context.Context, obj *storage.Version) error {
	return pgutils.Retry(ctx, func() error {
		return s.retryableUpsert(ctx, obj)
	})
}

func (s *storeImpl) retryableUpsert(ctx context.Context, obj *storage.Version) error {
	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_WRITE_ACCESS)
	if !scopeChecker.IsAllowed() {
		return sac.ErrResourceAccessDenied
	}

	tx, ctx, err := s.begin(ctx)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, deleteStmt); err != nil {
		if errTx := tx.Rollback(ctx); errTx != nil {
			return errors.Wrapf(errTx, "rolling back transaction due to: %v", err)
		}
		return err
	}

	if err := insertIntoVersions(ctx, tx, obj); err != nil {
		if errTx := tx.Rollback(ctx); errTx != nil {
			return errors.Wrapf(errTx, "rolling back transaction due to: %v", err)
		}
		return err
	}

	return tx.Commit(ctx)
}

// Get returns the object, if it exists from the store
func (s *storeImpl) Get(ctx context.Context) (*storage.Version, bool, error) {
	return pgutils.Retry3(ctx, func() (*storage.Version, bool, error) {
		return s.retryableGet(ctx)
	})
}

func (s *storeImpl) retryableGet(ctx context.Context) (*storage.Version, bool, error) {
	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_ACCESS)
	if !scopeChecker.IsAllowed() {
		return nil, false, nil
	}

	row := s.db.QueryRow(ctx, getStmt)
	var sequenceNum int
	var version string
	var minSequenceNum int
	var lastPersistedTime *time.Time
	if err := row.Scan(&sequenceNum, &version, &minSequenceNum, &lastPersistedTime); err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}

	msg := storage.Version{
		SeqNum:    int32(sequenceNum),
		Version:   version,
		MinSeqNum: int32(minSequenceNum),
	}

	if lastPersistedTime != nil {
		msg.LastPersisted = protoconv.MustConvertTimeToTimestamp(*lastPersistedTime)
	}
	return &msg, true, nil
}

func (s *storeImpl) begin(ctx context.Context) (*postgres.Tx, context.Context, error) {
	return postgres.GetTransaction(ctx, s.db)
}

// Delete removes the specified ID from the store
func (s *storeImpl) Delete(ctx context.Context) error {
	return pgutils.Retry(ctx, func() error {
		return s.retryableDelete(ctx)
	})
}

func (s *storeImpl) retryableDelete(ctx context.Context) error {
	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_WRITE_ACCESS)
	if !scopeChecker.IsAllowed() {
		return sac.ErrResourceAccessDenied
	}

	tx, ctx, err := s.begin(ctx)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, deleteStmt); err != nil {
		if errTx := tx.Rollback(ctx); errTx != nil {
			return errors.Wrapf(errTx, "rolling back transaction due to: %v", err)
		}
		return err
	}
	return tx.Commit(ctx)
}
