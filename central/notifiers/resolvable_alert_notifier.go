package notifiers

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// ResolvableAlertNotifier is the interface for notifiers that support the alert workflow
//
//go:generate mockgen-wrapper ResolvableAlertNotifier
type ResolvableAlertNotifier interface {
	AlertNotifier
	// AckAlert sends an acknowledgement of an alert.
	AckAlert(ctx context.Context, alert *storage.Alert) error
	// ResolveAlert resolves an alert.
	ResolveAlert(ctx context.Context, alert *storage.Alert) error
}
