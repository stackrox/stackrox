package notify

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
)

var log = logging.LoggerForModule()

const reconnectDelay = 5 * time.Second

// Handler is called for each notification received on a listened channel.
type Handler func(channel, payload string)

// LifecycleHooks synchronize consumers on every connection, including recovery.
// AfterListen runs after all LISTEN registrations have committed and before any
// notifications are dispatched. An error tears down the connection and retries.
// OnDisconnect runs on connection/setup failure and shutdown. Hooks run serially
// on the listener goroutine and must respect context cancellation.
type LifecycleHooks struct {
	AfterListen  func(context.Context) error
	OnDisconnect func(error)
}

// Listener listens on one or more PostgreSQL NOTIFY channels and dispatches
// notifications to a handler. It holds a dedicated connection outside the pool
// for the lifetime of the listener and automatically reconnects on failure.
type Listener struct {
	db       postgres.DB
	channels []string
	handler  Handler
	hooks    LifecycleHooks
}

// NewListener creates a Listener that will LISTEN on the given channels and
// call handler for each notification received.
func NewListener(db postgres.DB, handler Handler, channels ...string) *Listener {
	return NewListenerWithHooks(db, handler, LifecycleHooks{}, channels...)
}

// NewListenerWithHooks creates a listener with connection synchronization hooks.
func NewListenerWithHooks(db postgres.DB, handler Handler, hooks LifecycleHooks, channels ...string) *Listener {
	return &Listener{
		db:       db,
		channels: channels,
		handler:  handler,
		hooks:    hooks,
	}
}

// Listen blocks until ctx is cancelled, listening for notifications and
// dispatching them to the handler. It reconnects automatically on connection
// loss.
func (l *Listener) Listen(ctx context.Context) {
	for {
		err := l.listenLoop(ctx)
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		if l.hooks.OnDisconnect != nil {
			l.hooks.OnDisconnect(err)
		}
		if ctx.Err() != nil {
			return
		}
		log.Errorf("Notification listener error: %v, reconnecting in %v", err, reconnectDelay)
		select {
		case <-time.After(reconnectDelay):
		case <-ctx.Done():
			return
		}
	}
}

func (l *Listener) listenLoop(ctx context.Context) error {
	setupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := hijackConn(setupCtx, l.db)
	if err != nil {
		return fmt.Errorf("acquiring connection: %w", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = conn.Close(closeCtx)
	}()

	for _, ch := range l.channels {
		// Each Exec is an autocommitted statement, so the hook sees all LISTENs.
		if _, err := conn.Exec(setupCtx, "LISTEN "+pgx.Identifier{ch}.Sanitize()); err != nil {
			return fmt.Errorf("LISTEN %s: %w", ch, err)
		}
	}
	if l.hooks.AfterListen != nil {
		if err := l.hooks.AfterListen(ctx); err != nil {
			return fmt.Errorf("synchronizing notification listener: %w", err)
		}
	}

	log.Infof("Notification listener started on channels: %v", l.channels)

	for {
		notification, err := conn.WaitForNotification(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("waiting for notification: %w", err)
		}
		l.dispatchNotification(notification)
	}
}

func (l *Listener) dispatchNotification(n *pgconn.Notification) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Panic in notification handler for channel %q: %v\n%s", n.Channel, r, debug.Stack())
		}
	}()
	l.handler(n.Channel, n.Payload)
}

// hijackConn acquires a connection from the pool and permanently removes it
// via Hijack. The caller owns the returned *pgx.Conn and must close it.
func hijackConn(ctx context.Context, db postgres.DB) (*pgx.Conn, error) {
	poolConn, err := db.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	c, ok := poolConn.PgxPoolConn.(*pgxpool.Conn)
	if !ok {
		poolConn.Release()
		return nil, fmt.Errorf("cannot hijack connection (type: %T)", poolConn.PgxPoolConn)
	}
	return c.Hijack(), nil
}
