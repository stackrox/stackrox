package proxy

import (
	"context"
	"testing"

	"github.com/operator-framework/helm-operator-plugins/pkg/values"
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

func TestInjectProxyNoValueConflict(t *testing.T) {
	t.Setenv("HTTP_PROXY", "some.proxy.dns:443")

	proxyEnvVars := GetProxyEnvVars()
	proxyCustomizeTranslator := func(ctx context.Context, u *unstructured.Unstructured) (chartutil.Values, error) {
		customizeValue := translation.GetCustomize(&platform.CustomizeSpec{
			EnvVars: []corev1.EnvVar{
				{Name: "HTTP_PROXY", Value: "cr.proxy.value"},
			},
		})

		vb := translation.ValuesBuilder{}
		vb.AddChild("customize", customizeValue)

		return vb.Build()
	}

	injectTranslator := InjectProxyEnvVars(values.TranslatorFunc(proxyCustomizeTranslator), proxyEnvVars)
	values, err := injectTranslator.Translate(context.Background(), &unstructured.Unstructured{})
	if err != nil {
		t.Fatal(err)
	}

	envVars, err := values.Table("customize.envVars")
	require.NoError(t, err, "getting customize.envVars as YAML table")

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
