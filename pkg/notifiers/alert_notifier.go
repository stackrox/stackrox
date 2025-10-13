package notifiers

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// AlertNotifier is a notifier for active alerts
//
//go:generate mockgen-wrapper AlertNotifier
type AlertNotifier interface {
	Notifier
	// AlertNotify triggers the plugins to send a notification about an alert
	AlertNotify(ctx context.Context, alert *storage.Alert) error
}
