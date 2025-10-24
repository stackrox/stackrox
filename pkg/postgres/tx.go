package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	log = logging.LoggerForModule()
)

type txMode int

const (
	// the pgx.tx is used only inside a Postgres store
	original txMode = iota
	// the pgx.tx is used in both inside and outside of Postgres store
	// This tx wrapper is used inside the Postgres store
	inner
	// the pgx.tx is used in both inside and outside of Postgres store
	// This tx wrapper is used outside the Postgres store
	outer
)

// Tx wraps pgx.Tx
type Tx struct {
	pgx.Tx
	cancelFunc context.CancelFunc
	mode       txMode
}

// Exec wraps pgx.Tx Exec
func (t *Tx) Exec(ctx context.Context, sql string, args ...interface{}) (commandTag pgconn.CommandTag, err error) {
	ct, err := t.Tx.Exec(ctx, sql, args...)
	if err != nil {
		incQueryErrors(sql, err)
		return pgconn.CommandTag{}, err
	}
	return ct, err
}

// Query wraps pgx.Tx Query
func (t *Tx) Query(ctx context.Context, sql string, args ...interface{}) (*Rows, error) {
	rows, err := t.Tx.Query(ctx, sql, args...)
	if err != nil {
		incQueryErrors(sql, err)
		return nil, err
	}
	return &Rows{
		cancelFunc: func() {},
		query:      sql,
		Rows:       rows,
	}, nil
}

// QueryRow wraps pgx.Tx QueryRow
func (t *Tx) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return t.Tx.QueryRow(ctx, sql, args...)
}

// Commit wraps pgx.Tx Commit
func (t *Tx) Commit(ctx context.Context) error {
	defer t.cancelFunc()
	if t.mode == inner {
		return nil
	}

	if err := t.Tx.Commit(ctx); err != nil {
		incQueryErrors("commit", err)
		return err
	}
	return nil
}

// Rollback wraps pgx.Tx Rollback
func (t *Tx) Rollback(ctx context.Context) error {
	defer t.cancelFunc()

	if err := t.Tx.Rollback(ctx); err != nil {
		if t.mode == outer && errors.Is(err, pgx.ErrTxClosed) {
			// If this is an outer tx, the tx may have been rolled back
			// in its inner tx, so log and ignore this error
			log.Warnf("Failed to rollback outer tx: %v", err)
			return nil
		}
		incQueryErrors("rollback", err)
		return err
	}
	return nil
}

// UseInContext prepares tx to be passed to a store with context.
// this allows combine one or multiple operations before or after
// a store operation.
func (t *Tx) UseInContext() {
	switch t.mode {
	case original:
		t.mode = outer
	case inner:
		utils.Must(errors.New("it is not allowed to wrap a tx twice"))
	case outer:
		// This could be allowed in theory but I do not see the need, so disable it for simplicity.
		utils.Must(errors.New("it is not allowed to use one tx in two or more contexts"))
	}
}
