package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// TestEnsureBundleResourcesTypesAreCorrect exists to make accidental modifications of OrderedBundleResourcesTypes less
// likely. There is of course no protection against somebody intentionally modifying both lists.
func TestEnsureBundleResourcesTypesAreCorrect(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(t, OrderedBundleResourceTypes, []schema.GroupVersionKind{
		{Version: "v1", Kind: "Service"},
		{Version: "v1", Kind: "ServiceAccount"},
		{Version: "v1", Kind: "Secret"},
		{Version: "v1", Kind: "ConfigMap"},
		{Group: "admissionregistration.k8s.io", Version: "v1beta1", Kind: "ValidatingWebhookConfiguration"},
		{Group: "admissionregistration.k8s.io", Version: "v1", Kind: "ValidatingWebhookConfiguration"},
		{Group: "networking.k8s.io", Version: "v1", Kind: "NetworkPolicy"},
		{Group: "networking.istio.io", Version: "v1alpha3", Kind: "DestinationRule"},
		{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"},
		{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"},
		{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"},
		{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"},
		{Group: "security.openshift.io", Version: "v1", Kind: "SecurityContextConstraints"},
		{Group: "policy", Version: "v1beta1", Kind: "PodSecurityPolicy"},
		{Group: "apps", Version: "v1", Kind: "DaemonSet"},
		{Group: "apps", Version: "v1", Kind: "Deployment"},
		{Group: "monitoring.coreos.com", Version: "v1", Kind: "ServiceMonitor"},
		{Group: "monitoring.coreos.com", Version: "v1", Kind: "PrometheusRule"},
	})
}
