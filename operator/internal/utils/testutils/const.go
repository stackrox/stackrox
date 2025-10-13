package testutils

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

const (
	// TestNamespace is the name of a test namespace.
	TestNamespace = "testns"
)

// ValidClusterVersion represents Openshift custom resource for cluster version.
var ValidClusterVersion = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       "ClusterVersion",
		"apiVersion": "config.openshift.io/v1",
		"metadata": map[string]interface{}{
			"name": "version",
		},
	},
}
