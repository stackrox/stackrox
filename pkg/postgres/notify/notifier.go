package notify

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/pkg/postgres"
)

// Notify sends a notification on the given channel with the given payload.
// The notification is delivered to any connections currently LISTENing on the channel.
func Notify(ctx context.Context, db postgres.DB, channel, payload string) error {
	_, err := db.Exec(ctx, "SELECT pg_notify($1, $2)", channel, payload)
	if err != nil {
		return fmt.Errorf("pg_notify on channel %q: %w", channel, err)
	}
	return nil
}
