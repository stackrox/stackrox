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
func InjectProxyEnvVars(translator values.Translator, proxyEnv map[string]string) values.Translator {
	return values.TranslatorFunc(func(ctx context.Context, obj *unstructured.Unstructured) (chartutil.Values, error) {
		vals, err := translator.Translate(ctx, obj)
		if err != nil {
			return nil, err
		}

		proxyVals, _ := getProxyConfigHelmValues(obj, proxyEnv) // ignore errors for now

		mergedVals := chartutil.CoalesceTables(vals, proxyVals)
		mergedVals = delValueFromIfValueExists(mergedVals)

		return mergedVals, nil
	})
}

// delValueFromIfValueExists deletes the valueFrom key from customize.envVars entries if both value and valueFrom key exists
// returns the unmodified values in case of error
func delValueFromIfValueExists(values chartutil.Values) chartutil.Values {
	envVarsMap, err := values.Table("customize.envVars")
	if err != nil {
		return values
	}

	for envVarName := range envVarsMap {
		envVar, err := envVarsMap.Table(envVarName)
		if err != nil {
			return values
		}

		_, hasValue := envVar["value"]
		_, hasValueFrom := envVar["valueFrom"]

		if hasValue && hasValueFrom {
			delete(envVar, "valueFrom")
		}
	}

	return values
}
