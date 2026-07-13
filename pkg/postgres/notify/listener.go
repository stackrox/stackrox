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

// Listener listens on one or more PostgreSQL NOTIFY channels and dispatches
// notifications to a handler. It holds a dedicated connection outside the pool
// for the lifetime of the listener and automatically reconnects on failure.
type Listener struct {
	db       postgres.DB
	channels []string
	handler  Handler
}

// NewListener creates a Listener that will LISTEN on the given channels and
// call handler for each notification received.
func NewListener(db postgres.DB, handler Handler, channels ...string) *Listener {
	return &Listener{
		db:       db,
		channels: channels,
		handler:  handler,
	}
}

// Listen blocks until ctx is cancelled, listening for notifications and
// dispatching them to the handler. It reconnects automatically on connection
// loss.
func (l *Listener) Listen(ctx context.Context) {
	for {
		if err := l.listenLoop(ctx); err != nil {
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
}

func (l *Listener) listenLoop(ctx context.Context) error {
	conn, err := hijackConn(ctx, l.db)
	if err != nil {
		return fmt.Errorf("acquiring connection: %w", err)
	}
	defer func() { _ = conn.Close(context.Background()) }()

	for _, ch := range l.channels {
		if _, err := conn.Exec(ctx, "LISTEN "+pgx.Identifier{ch}.Sanitize()); err != nil {
			return fmt.Errorf("LISTEN %s: %w", ch, err)
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
