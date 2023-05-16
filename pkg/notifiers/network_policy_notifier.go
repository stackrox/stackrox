package notifiers

import "context"

// NetworkPolicyNotifier is for sending network policies
type NetworkPolicyNotifier interface {
	Notifier
	// NetworkPolicyYAMLNotify triggers the plugins to send a notification about a network policy yaml
	NetworkPolicyYAMLNotify(ctx context.Context, yaml string, clusterName string) error
}
