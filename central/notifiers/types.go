package notifiers

import (
	"github.com/stackrox/rox/generated/storage"
)

// Notifier interface defines the contract that all plugins must satisfy
type Notifier interface {
	// AlertNotify triggers the plugins to send a notification about an alert
	AlertNotify(alert *storage.Alert) error
	// YamlNotify triggers the plugins to send a notification about a network policy yaml
	NetworkPolicyYAMLNotify(yaml string, clusterName string) error
	// ProtoNotifier gets the proto version of the notifier
	ProtoNotifier() *storage.Notifier
	// Test sends a test message
	Test() error
}
