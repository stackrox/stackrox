package notifiers

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Notifier is the base notifier that all types of notifiers must implement
type Notifier interface {
	// ProtoNotifier gets the proto version of the notifier
	ProtoNotifier() *storage.Notifier
	// Test sends a test message
	Test() error
}

// AlertNotifier is a notifier for active alerts
type AlertNotifier interface {
	Notifier
	// AlertNotify triggers the plugins to send a notification about an alert
	AlertNotify(alert *storage.Alert) error
}

// ResolvableAlertNotifier is the interface for notifiers that support the alert workflow
type ResolvableAlertNotifier interface {
	AlertNotifier
	// AckAlert sends an acknowledges an alert
	AckAlert(alert *storage.Alert) error
	// ResolveAlert resolves an alert
	ResolveAlert(alert *storage.Alert) error
}

// AuditNotifier is the notifier for audit logs
type AuditNotifier interface {
	Notifier
	// SendAuditMessage sends an audit message
	SendAuditMessage(msg *v1.Audit_Message) error
	AuditLoggingEnabled() bool
}

// NetworkPolicyNotifier is for sending network policies
type NetworkPolicyNotifier interface {
	Notifier
	// NetworkPolicyYAMLNotify triggers the plugins to send a notification about a network policy yaml
	NetworkPolicyYAMLNotify(yaml string, clusterName string) error
}
