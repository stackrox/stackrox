package proxy

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/stackrox/rox/operator/pkg/values/translation"
	"github.com/stackrox/rox/pkg/k8sutil"
	"helm.sh/helm/v3/pkg/chartutil"
)

func getProxyConfigHelmValues(obj k8sutil.Object, proxyEnvVars map[string]string) chartutil.Values {
	if len(proxyEnvVars) == 0 {
		return nil
	}

	secretName := getProxyEnvSecretName(obj)

	envVarsMap := map[string]interface{}{}
	for envVarName := range proxyEnvVars {
		envVarsMap[envVarName] = map[string]interface{}{
			"valueFrom": map[string]interface{}{
				"secretKeyRef": map[string]interface{}{
					"key":  envVarName,
					"name": secretName,
				},
			},
		}
	}

	return chartutil.Values{
		"customize": map[string]interface{}{
			"envVars": envVarsMap,
		},
	}
}

// NewProxyEnvVarsInjector returns an object which injects proxy env vars into enriched chart values.
func NewProxyEnvVarsInjector(proxyEnv map[string]string, log logr.Logger) *proxyEnvVarsInjector {
	return &proxyEnvVarsInjector{
		proxyEnv: proxyEnv,
		log:      log,
	}
}

type proxyEnvVarsInjector struct {
	proxyEnv map[string]string
	log      logr.Logger
}

var _ translation.Enricher = &proxyEnvVarsInjector{}

// Enrich injects proxy configuration environment variables.
func (i *proxyEnvVarsInjector) Enrich(_ context.Context, obj k8sutil.Object, vals chartutil.Values) (chartutil.Values, error) {
	proxyVals := getProxyConfigHelmValues(obj, i.proxyEnv)

	mergedVals := chartutil.CoalesceTables(vals, proxyVals)

	mergedVals, conflicts := deleteValueFromIfValueExists(mergedVals)
	if len(conflicts) > 0 {
		err := fmt.Errorf("conflicts: %s for %s/%s", conflicts, obj.GetNamespace(), obj.GetName())
		i.log.Error(err, "injecting proxy env vars")
	}

	return mergedVals, nil
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
