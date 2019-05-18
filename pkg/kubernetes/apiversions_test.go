package kubernetes

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsNativeAPI(t *testing.T) {
	cases := []struct {
		apiVersion string
		isTracked  bool
	}{
		{
			isTracked: true,
		},
		{
			apiVersion: "v1",
			isTracked:  true,
		},
		{
			apiVersion: "policy/v1beta1",
			isTracked:  true,
		},
		{
			apiVersion: "rbac.authorization.k8s.io/v1",
			isTracked:  true,
		},
		{
			apiVersion: "serving.knative.dev/v1alpha1",
			isTracked:  false,
		},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf(c.apiVersion), func(t *testing.T) {
			assert.Equal(t, IsNativeAPI(c.apiVersion), c.isTracked)
		})
	}
}
