package proxy

import (
	"testing"

	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	testObj = &platform.Central{
		TypeMeta: metav1.TypeMeta{
			APIVersion: platform.CentralGVK.GroupVersion().String(),
			Kind:       platform.CentralGVK.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-central",
		},
	}
)

func TestGetProxyConfigHelmValues_EmptyEnv(t *testing.T) {
	vals, err := getProxyConfigHelmValues(testObj, nil)
	assert.NoError(t, err)
	assert.Empty(t, vals)
}

func TestGetProxyConfigHelmValues_WithValues(t *testing.T) {
	env := map[string]string{
		"http_proxy": "http://my.proxy",
		"NO_PROXY":   "127.0.0.1/8",
	}

	vals, err := getProxyConfigHelmValues(testObj, env)
	assert.NoError(t, err)

	expectedVals, err := chartutil.ReadValues([]byte(`
customize:
  envVars:
    http_proxy:
      valueFrom:
       secretKeyRef:
         name: central-test-central-proxy-env
         key: http_proxy
    NO_PROXY:
      valueFrom:
        secretKeyRef:
          name: central-test-central-proxy-env
          key: NO_PROXY`))

	require.NoError(t, err)
	assert.Equal(t, expectedVals, vals)
}
