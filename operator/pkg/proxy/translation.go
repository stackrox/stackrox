package proxy

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/values"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/utils"
	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func getProxyConfigEnvVars(obj k8sutil.Object, proxyEnvVars map[string]string) map[string]interface{} {
	if len(proxyEnvVars) == 0 {
		return nil
	}

	secretName := getProxyEnvSecretName(obj)

	envVarsMap := map[string]interface{}{}
	for envVarName := range proxyEnvVars {
		envVarsMap[envVarName] = map[string]interface{}{
			"valueFrom": map[string]interface{}{
				"secretKeyRef": map[string]interface{}{
					"name": secretName,
					"key":  envVarName,
				},
			},
		}
	}

	return envVarsMap
}

// InjectProxyEnvVars wraps a Translator to inject proxy configuration environment variables.
func InjectProxyEnvVars(translator values.Translator, proxyEnv map[string]string, log logr.Logger) values.Translator {
	return values.TranslatorFunc(func(ctx context.Context, obj *unstructured.Unstructured) (chartutil.Values, error) {
		vals, err := translator.Translate(ctx, obj)
		if err != nil {
			return nil, err
		}

		proxyEnvVars := getProxyConfigEnvVars(obj, proxyEnv)
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
