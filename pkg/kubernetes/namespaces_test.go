//go:build test_all

package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSystemNamespace(t *testing.T) {
	cases := []struct {
		namespace string
		system    bool
	}{
		{
			namespace: "kube-system",
			system:    true,
		},
		{
			namespace: "kube-public",
			system:    true,
		},
		{
			namespace: "openshift-kube-apiserver",
			system:    true,
		},
		{
			namespace: "openshift",
			system:    true,
		},
		{
			namespace: "istio-system",
			system:    true,
		},
		{
			namespace: "stackrox",
			system:    false,
		},
		{
			namespace: "default",
			system:    false,
		},
		{
			namespace: "openshift-operators",
			system:    false,
		},
	}
	for _, c := range cases {
		t.Run(c.namespace, func(t *testing.T) {
			assert.Equal(t, c.system, IsSystemNamespace(c.namespace))
		})
	}
}
