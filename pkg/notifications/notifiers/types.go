package notifiers

import "github.com/stackrox/rox/generated/api/v1"

// Notifier interface defines the contract that all plugins must satisfy
type Notifier interface {
	// AlertNotify triggers the plugins to send a notification about an alert
	AlertNotify(alert *v1.Alert) error
	// BenchmarkNotify triggers the plugins to send a notification about a benchmark
	BenchmarkNotify(schedule *v1.BenchmarkSchedule) error
	// ProtoNotifier gets the proto version of the notifier
	ProtoNotifier() *v1.Notifier
	// Test sends a test message
	Test() error
}
