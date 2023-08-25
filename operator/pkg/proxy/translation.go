package proxy

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/values"
	"github.com/stackrox/rox/pkg/k8sutil"
	"helm.sh/helm/v3/pkg/chartutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func getProxyConfigHelmValues(obj k8sutil.Object, proxyEnvVars map[string]string) (chartutil.Values, error) {
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

	return chartutil.Values{
		"customize": map[string]interface{}{
			"envVars": envVarsMap,
		},
	}, nil
}

// InjectProxyEnvVars wraps a Translator to inject proxy configuration environment variables.
func InjectProxyEnvVars(translator values.Translator, proxyEnv map[string]string, log logr.Logger) values.Translator {
	return values.TranslatorFunc(func(ctx context.Context, obj *unstructured.Unstructured) (chartutil.Values, error) {
		vals, err := translator.Translate(ctx, obj)
		if err != nil {
			return nil, err
		}

		proxyVals, _ := getProxyConfigHelmValues(obj, proxyEnv) // ignore errors for now

		mergedVals := chartutil.CoalesceTables(vals, proxyVals)

		mergedVals, conflicts := deleteValueFromIfValueExists(mergedVals)
		if len(conflicts) > 0 {
			err := fmt.Errorf("conflicts: %s for %s/%s", conflicts, obj.GetNamespace(), obj.GetName())
			log.Error(err, "injecting proxy env vars")
		}

		return mergedVals, nil
	})
}

// deleteValueFromIfValueExists deletes the valueFrom key from customize.envVars entries
// if both value and valueFrom key exist. Returns the unmodified values in case of error in accessing values.
// This function was introduced to fix the bug documented in ROX-18477
func deleteValueFromIfValueExists(values chartutil.Values) (chartutil.Values, []string) {
	envVarsMap, err := values.Table("customize.envVars")
	conflicts := []string{}

	if err != nil {
		return values, conflicts
	}

	for envVarName := range envVarsMap {
		envVar, err := envVarsMap.Table(envVarName)
		if err != nil {
			return values, conflicts
		}

		_, hasValue := envVar["value"]
		_, hasValueFrom := envVar["valueFrom"]

		if hasValue && hasValueFrom {
			delete(envVar, "valueFrom")
			valueConflict := fmt.Sprintf("env var: %s, has both value and valueFrom set", envVarName)
			conflicts = append(conflicts, valueConflict)
		}
	}

	return values, conflicts
}
