package preflight

import (
	"testing"

	"github.com/stackrox/rox/pkg/k8sutil/k8sobjects"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestNamespaceExceptionMatching(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		object    *k8sobjects.ObjectRef
		isAllowed bool
	}{
		"ServiceMonitor in openshift-monitoring": {
			object: &k8sobjects.ObjectRef{
				GVK: schema.GroupVersionKind{Kind: "ServiceMonitor"}, Namespace: namespaces.OpenShiftMonitoring,
			},
			isAllowed: true,
		},
		"PrometheusRule in openshift-monitoring": {
			object: &k8sobjects.ObjectRef{
				GVK: schema.GroupVersionKind{Kind: "PrometheusRule"}, Namespace: namespaces.OpenShiftMonitoring,
			},
			isAllowed: true,
		},
		"RoleBinding in kube-system": {
			object: &k8sobjects.ObjectRef{
				GVK: schema.GroupVersionKind{Kind: "RoleBinding"}, Namespace: namespaces.KubeSystem,
			},
			isAllowed: true,
		},
		"ServiceMonitor in kube-system": {
			object: &k8sobjects.ObjectRef{
				GVK: schema.GroupVersionKind{Kind: "ServiceMonitor"}, Namespace: namespaces.KubeSystem,
			},
			isAllowed: false,
		},
		"PrometheusRule in stackrox": {
			object: &k8sobjects.ObjectRef{
				GVK: schema.GroupVersionKind{Kind: "PrometheusRule"}, Namespace: namespaces.StackRox,
			},
			isAllowed: true,
		},
		"Deployment in stackrox": {
			object: &k8sobjects.ObjectRef{
				GVK: schema.GroupVersionKind{Kind: "Deployment"}, Namespace: namespaces.StackRox,
			},
			isAllowed: true,
		},
		"Deployment in kube-system": {
			object: &k8sobjects.ObjectRef{
				GVK: schema.GroupVersionKind{Kind: "Deployment"}, Namespace: namespaces.KubeSystem,
			},
			isAllowed: false,
		},
		"ClusterRoleBinding": {
			object: &k8sobjects.ObjectRef{
				GVK: schema.GroupVersionKind{Kind: "ClusterRoleBinding"}, Namespace: "",
			},
			isAllowed: true,
		},
	}
	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.isAllowed, namespaceAllowed(tt.object))
		})
	}
}
