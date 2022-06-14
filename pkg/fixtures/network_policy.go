package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
)

// GetYAML returns a network policy yaml.
func GetYAML() string {
	return `kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
	name: allow-traffic-from-apps-using-multiple-selectors
spec:
	podSelector:
		matchLabels:
			app: web
			role: db
		ingress:
			- from:
				- podSelector:
					matchLabels:
						app: bookstore
						role: search
				- podSelector:
					matchLabels:
						app: bookstore
						role: api`
}

// GetNetworkPolicy returns a network policy.
func GetNetworkPolicy() *storage.NetworkPolicy {
	return GetScopedNetworkPolicy("network-policy-id", "cluster-id", "namespace")
}

// GetScopedNetworkPolicy returns a network policy holding the provided scope information.
func GetScopedNetworkPolicy(id string, clusterID string, namespace string) *storage.NetworkPolicy {
	return &storage.NetworkPolicy{
		Id:          id,
		Name:        "network-policy-name",
		ClusterId:   clusterID,
		ClusterName: "",
		Namespace:   namespace,
		Labels:      nil,
		Annotations: nil,
		Spec: &storage.NetworkPolicySpec{
			PodSelector: nil,
			Ingress:     nil,
			Egress:      nil,
			PolicyTypes: nil,
		},
		Yaml:       "",
		ApiVersion: "",
		Created:    nil,
	}
}
