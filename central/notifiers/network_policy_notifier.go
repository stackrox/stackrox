package notifiers

// NetworkPolicyNotifier is for sending network policies
type NetworkPolicyNotifier interface {
	Notifier
	// NetworkPolicyYAMLNotify triggers the plugins to send a notification about a network policy yaml
	NetworkPolicyYAMLNotify(yaml string, clusterName string) error
}
