package proxy

import (
	"context"

	"github.com/operator-framework/helm-operator-plugins/pkg/values"
	"github.com/stackrox/rox/pkg/k8sutil"
	"helm.sh/helm/v3/pkg/chartutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func getProxyConfigEnvVars(obj k8sutil.Object, proxyEnvVars map[string]string) (map[string]interface{}, error) {
	if len(proxyEnvVars) == 0 {
		return nil, nil
	}

	secretName := getProxyEnvSecretName(obj)

	envVarsMap := map[string]interface{}{}
	for envVarName := range proxyEnvVars {
		src := v1.EnvVarSource{
			SecretKeyRef: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: secretName,
				},
				Key: envVarName,
			},
		}
		uSrc, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&src)
		if err != nil {
			return nil, err
		}
		envVarsMap[envVarName] = map[string]interface{}{
			"valueFrom": uSrc,
		}
	}

	return envVarsMap, nil
}

// InjectProxyEnvVars wraps a Translator to inject proxy configuration environment variables.
func InjectProxyEnvVars(translator values.Translator, proxyEnv map[string]string) values.Translator {
	return values.TranslatorFunc(func(ctx context.Context, obj *unstructured.Unstructured) (chartutil.Values, error) {
		vals, err := translator.Translate(ctx, obj)
		if err != nil {
			return nil, err
		}

		proxyEnvVars, _ := getProxyConfigEnvVars(obj, proxyEnv) // ignore errors for now
		if len(proxyEnvVars) == 0 {
			return vals, nil
		}

		// We must only set environment variables which are not already set via the CR. Otherwise, we might end up with
		// an invalid deployment spec, where env entries have both a `value` and `valueFrom` set.

		// Make sure customize.envVars section is present
		vals = chartutil.CoalesceTables(vals, map[string]interface{}{
			"customize": map[string]interface{}{
				"envVars": map[string]interface{}{},
			},
		})

		envVarVals, err := vals.Table("customize.envVars")
		if err != nil {
			return vals, nil // give up on injecting env vars, something is off
		}

		for envVarName, envVarSpec := range proxyEnvVars {
			if _, ok := envVarVals[envVarName]; ok {
				continue // env var already set
			}
			envVarVals[envVarName] = envVarSpec
		}

		return vals, nil
	})
}
