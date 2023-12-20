package proxy

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/values/translation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	vals := getProxyConfigHelmValues(testObj, nil)
	assert.Empty(t, vals)
}

func TestGetProxyConfigHelmValues_WithValues(t *testing.T) {
	env := map[string]string{
		"http_proxy": "http://my.proxy",
		"NO_PROXY":   "127.0.0.1/8",
	}

	vals := getProxyConfigHelmValues(testObj, env)

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

func TestInjectProxyNoValueConflict(t *testing.T) {
	t.Setenv("HTTP_PROXY", "some.proxy.dns:443")

	customizeValue := translation.GetCustomize(&platform.CustomizeSpec{
		EnvVars: []corev1.EnvVar{
			{Name: "HTTP_PROXY", Value: "cr.proxy.value"},
		},
	})
	baseValues, err := (&translation.ValuesBuilder{}).AddChild("customize", customizeValue).Build()
	require.NoError(t, err)

	injector := NewProxyEnvVarsInjector(GetProxyEnvVars(), logr.New(nil))
	values, err := injector.Enrich(context.Background(), &unstructured.Unstructured{}, baseValues)
	require.NoError(t, err, "translating values")

	envVars, err := values.Table("customize.envVars")
	require.NoError(t, err, "getting customize.envVars as YAML table")

	require.Len(t, envVars, 1)
	for envVarName := range envVars {
		envVar, err := envVars.Table(envVarName)
		require.NoErrorf(t, err, "getting %s as YAML table", envVarName)

		_, hasValue := envVar["value"]
		_, hasValueFrom := envVar["valueFrom"]

		if hasValue && hasValueFrom {
			t.Fatalf("env var: %s has value conflict. Both value and value from set: %v", envVarName, envVars[envVarName])
		}
	}
}
