package proxy

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/values"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/utils"
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
func InjectProxyEnvVars(translator values.Translator, proxyEnv map[string]string, log logr.Logger) values.Translator {
	return values.TranslatorFunc(func(ctx context.Context, obj *unstructured.Unstructured) (chartutil.Values, error) {
		vals, err := translator.Translate(ctx, obj)
		if err != nil {
			return nil, err
		}

		proxyEnvVars, err := getProxyConfigEnvVars(obj, proxyEnv)
		if err != nil {
			// Simply log the error, we do not want to fail reconciliation based on this (the check for
			// len(proxyEnvVars) == 0) should catch a complete failure).
			// While this log can be spammy (emitted on every reconciliation), an error here is extremely unlikely
			// and thus we deem this acceptable.
			log.Error(err, "could not determine proxy environment variables", "gvk", obj.GroupVersionKind(), "namespace", obj.GetNamespace(), "name", obj.GetName())
		}

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
			// This shouldn't happen due to the CoalesceTables call above.
			utils.Should(errors.Wrap(err, "failed to look up customize.envVars table"))
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
