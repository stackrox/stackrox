package postgres

import (
	"context"
	"errors"
	"sync"

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
	lifecycle  *txLifecycle
}

type txLifecycle struct {
	mu        sync.Mutex
	finished  bool
	callbacks []func()
}

func (l *txLifecycle) afterFinish(fn func()) {
	if fn == nil {
		return
	}
	l.mu.Lock()
	if !l.finished {
		l.callbacks = append(l.callbacks, fn)
		l.mu.Unlock()
		return
	}
	l.mu.Unlock()
	fn()
}

func (l *txLifecycle) finish() {
	l.mu.Lock()
	if l.finished {
		l.mu.Unlock()
		return
	}
	l.finished = true
	callbacks := l.callbacks
	l.callbacks = nil
	l.mu.Unlock()
	for _, callback := range callbacks {
		callback()
	}
}

// AfterFinish registers a callback invoked once the outer transaction commits
// or rolls back. It is intended for state maintained outside PostgreSQL, such
// as retained caches, that must not be refreshed before the transaction ends.
func (t *Tx) AfterFinish(fn func()) {
	if t.lifecycle == nil {
		fn()
		return
	}
	t.lifecycle.afterFinish(fn)
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

// Commit wraps pgx.Tx Commit but is a NOOP for inner transaction
func (t *Tx) Commit(ctx context.Context) error {
	// Ensure commit completes even if parent context is cancelled
	ctx = context.WithoutCancel(ctx)
	defer func() {
		if t.cancelFunc != nil {
			t.cancelFunc()
		}
	}()
	if t.mode == inner {
		return nil
	}

	if err := t.Tx.Commit(ctx); err != nil {
		incQueryErrors("commit", err)
		if t.lifecycle != nil {
			t.lifecycle.finish()
		}
		return err
	}
	if t.lifecycle != nil {
		t.lifecycle.finish()
	}
	return nil
}

// Rollback wraps pgx.Tx Rollback safe to use even if inner transaction was reverted.
// In case Rollback or Commit might have the same effect (e.g. read only transaction)
// prefer Commit.
func (t *Tx) Rollback(ctx context.Context) error {
	// Ensure rollback completes even if parent context is cancelled
	ctx = context.WithoutCancel(ctx)
	defer func() {
		if t.cancelFunc != nil {
			t.cancelFunc()
		}
	}()

	if err := t.Tx.Rollback(ctx); err != nil {
		if t.mode == outer && errors.Is(err, pgx.ErrTxClosed) {
			// If this is an outer tx, the tx may have been rolled back
			// in its inner tx, so log and ignore this error
			log.Warnf("Failed to rollback outer tx: %v", err)
			if t.lifecycle != nil {
				t.lifecycle.finish()
			}
			return nil
		}
		incQueryErrors("rollback", err)
		if t.lifecycle != nil {
			t.lifecycle.finish()
		}
		return err
	}
	if t.lifecycle != nil {
		t.lifecycle.finish()
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

// GetTransaction returns a new transaction or one from context.
func GetTransaction(ctx context.Context, db DB) (*Tx, context.Context, error) {
	if tx, ok := TxFromContext(ctx); ok {
		return &Tx{
			Tx:         tx.Tx,
			cancelFunc: tx.cancelFunc,
			mode:       inner,
			lifecycle:  tx.lifecycle,
		}, ctx, nil
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	ctxWithTx := ContextWithTx(ctx, tx)
	return tx, ctxWithTx, nil
}

// FinishReadOnlyTransaction commits the transaction and log error.
// Since context might be already done it uses its own background context.
func FinishReadOnlyTransaction(tx interface {
	Commit(ctx context.Context) error
}) {
	if err := tx.Commit(context.Background()); err != nil {
		log.Errorf("failed to commit transaction: %v", err)
	}
}
