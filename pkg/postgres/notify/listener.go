package notify

import (
	"context"
	"fmt"
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
// notifications to a handler. It holds a dedicated connection from the pool
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
	conn, err := l.db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	pgxConn, err := underlyingConn(conn)
	if err != nil {
		return err
	}

	for _, ch := range l.channels {
		if _, err := pgxConn.Exec(ctx, "LISTEN "+ch); err != nil {
			return fmt.Errorf("LISTEN %s: %w", ch, err)
		}
	}

	log.Infof("Notification listener started on channels: %v", l.channels)

	for {
		notification, err := pgxConn.WaitForNotification(ctx)
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
			log.Errorf("Panic in notification handler for channel %q: %v", n.Channel, r)
		}
	}()
	l.handler(n.Channel, n.Payload)
}

// underlyingConn extracts the *pgx.Conn from the pool wrapper.
// pgxpool.Conn exposes .Conn() which returns the underlying *pgx.Conn
// that has WaitForNotification.
func underlyingConn(conn *postgres.Conn) (*pgx.Conn, error) {
	if c, ok := conn.PgxPoolConn.(*pgxpool.Conn); ok {
		return c.Conn(), nil
	}
	return nil, fmt.Errorf("cannot extract *pgx.Conn from connection wrapper (type: %T)", conn.PgxPoolConn)
}
