package proxy

import (
	"context"
	"testing"

	"github.com/operator-framework/helm-operator-plugins/pkg/values"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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

	testObjUnstructured = func() *unstructured.Unstructured {
		obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(testObj)
		utils.CrashOnError(err)
		return &unstructured.Unstructured{
			Object: obj,
		}
	}
)

func TestGetProxyConfigEnvVars_EmptyEnv(t *testing.T) {
	vals, err := getProxyConfigEnvVars(testObj, nil)
	assert.NoError(t, err)
	assert.Empty(t, vals)
}

func TestGetProxyConfigEnvVars_WithValues(t *testing.T) {
	env := map[string]string{
		"http_proxy": "http://my.proxy",
		"NO_PROXY":   "127.0.0.1/8",
	}

	vals, err := getProxyConfigEnvVars(testObj, env)
	assert.NoError(t, err)

	expectedVals, err := chartutil.ReadValues([]byte(`
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
	assert.EqualValues(t, expectedVals, vals)
}

func TestInjectProxyEnvVars(t *testing.T) {
	env := map[string]string{
		"http_proxy": "http://my.proxy",
		"NO_PROXY":   "127.0.0.1/8",
	}

	vals, err := InjectProxyEnvVars(values.TranslatorFunc(func(_ context.Context, _ *unstructured.Unstructured) (chartutil.Values, error) {
		return chartutil.ReadValues([]byte(`
foo:
  bar: baz`))
	}), env).Translate(context.Background(), testObjUnstructured())
	assert.NoError(t, err)

	expectedVals, err := chartutil.ReadValues([]byte(`
foo:
  bar: baz
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
	assert.EqualValues(t, expectedVals, vals)
}

func TestInjectProxyEnvVars_NoConflict(t *testing.T) {
	env := map[string]string{
		"http_proxy": "http://my.proxy",
		"NO_PROXY":   "127.0.0.1/8",
	}

	vals, err := InjectProxyEnvVars(values.TranslatorFunc(func(_ context.Context, _ *unstructured.Unstructured) (chartutil.Values, error) {
		return chartutil.ReadValues([]byte(`
foo:
  bar: baz
customize:
  envVars:
    SOME_VAR: foo
    ANOTHER_VAR:
      value: bar`))
	}), env).Translate(context.Background(), testObjUnstructured())
	assert.NoError(t, err)

	expectedVals, err := chartutil.ReadValues([]byte(`
foo:
  bar: baz
customize:
  envVars:
    SOME_VAR: foo
    ANOTHER_VAR:
      value: bar
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
	assert.EqualValues(t, expectedVals, vals)
}

func TestInjectProxyEnvVars_WithConflict(t *testing.T) {
	env := map[string]string{
		"http_proxy": "http://my.proxy",
		"NO_PROXY":   "127.0.0.1/8",
		"ALL_PROXY":  "http://my.other.proxy",
	}

	vals, err := InjectProxyEnvVars(values.TranslatorFunc(func(_ context.Context, _ *unstructured.Unstructured) (chartutil.Values, error) {
		return chartutil.ReadValues([]byte(`
foo:
  bar: baz
customize:
  envVars:
    SOME_VAR: foo
    http_proxy:
      value: bar
    NO_PROXY: baz`))
	}), env).Translate(context.Background(), testObjUnstructured())
	assert.NoError(t, err)

	expectedVals, err := chartutil.ReadValues([]byte(`
foo:
  bar: baz
customize:
  envVars:
    SOME_VAR: foo
    http_proxy:
      value: bar
    NO_PROXY: baz
    ALL_PROXY:
      valueFrom:
        secretKeyRef:
          name: central-test-central-proxy-env
          key: ALL_PROXY`))

	require.NoError(t, err)
	assert.EqualValues(t, expectedVals, vals)
}
