package derivelocalvalues

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/roxctl/helm/internal/common"
	"gopkg.in/yaml.v3"
)

func deriveLocalValuesForChart(namespace, chartName, input, output string) error {
	var err error
	context, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()
	switch chartName {
	case common.ChartCentralServices:
		err = deriveLocalValuesForCentralServices(context, namespace, input, output)
	default:
		fmt.Fprintf(os.Stderr, "Deriving local values for chart %q is currently unsupported.", chartName)
		fmt.Fprintf(os.Stderr, "Supported charts: %s", common.PrettyChartNameList)
		err = errors.Errorf("unsupported chart %q", chartName)
	}

	return err
}

// Implementation for command `helm derive-local-values`.
func deriveLocalValuesForCentralServices(context context.Context, namespace, input, output string) error {
	var k8s k8sObjectDescription

	if input == "" {
		// Connect to running cluster and retrieve K8s resource definitions from there.
		k8sLive, err := newLiveK8sObjectDescription()
		if err != nil {
			return errors.Wrap(err, "connecting to configured Kubernetes cluster")
		}
		k8s = newK8sObjectDescription(k8sLive)
	} else {
		// Retrieve K8s resource definitions from local YAML files.
		k8sLocal, err := newLocalK8sObjectDescription(input)
		if err != nil {
			return errors.Wrapf(err, "retrieving Kubernetes resource definitions from %q", input)
		}
		k8s = newK8sObjectDescription(k8sLocal)
	}

	// Note regarding custom metadata (annotations, labels and env vars): We make it easy for us:
	// we simply retrieve the metadata from the central deployment and assume that any custom metadata
	// on that resource is to be used globally for all StackRox resources.

	var scannerConfig map[string]interface{}
	if k8s.Exists(context, "deployment", "scanner") {
		scannerConfig = map[string]interface{}{
			"replicas": k8s.evaluateToInt64(context, "deployment", "scanner", `{.spec.replicas}`, 3),
			"autoscaling": k8s.evaluateToSubObject(context, "hpa", "scanner", `{.spec}`, []string{"minReplicas", "maxReplicas"},
				map[string]interface{}{"disable": true}),
			"resources": k8s.evaluateToObject(context, "deployment", "scanner",
				`{.spec.template.spec.containers[?(@.name == "scanner")].resources}`, nil),
			"image": map[string]interface{}{
				"registry": extractImageRegistry(k8s.evaluateToString(context, "deployment", "scanner",
					`{.spec.template.spec.containers[?(@.name == "scanner")].image}`, ""), "scanner"),
			},
			"dbImage": map[string]interface{}{
				"registry": extractImageRegistry(k8s.evaluateToString(context, "deployment", "scanner-db",
					`{.spec.template.spec.containers[?(@.name == "db")].image}`, ""), "scanner-db"),
			},
			"dbResources": k8s.evaluateToObject(context, "deployment", "scanner-db",
				`{.spec.template.spec.containers[?(@.name == "db")].resources}`, nil),
		}
	} else {
		scannerConfig = map[string]interface{}{
			"disable": true,
		}
	}

	m := map[string]interface{}{
		"licenseKey": k8s.evaluateToStringP(context, "secret", "central-license", `{.data['license\.lic']}`),
		// "image": We do not specify a global registry,
		// instead we only specify central- and scanner-specific registries.
		"env": map[string]interface{}{
			"offlineMode": k8s.evaluateToString(context, "deployment", "central",
				`{.spec.template.spec.containers[?(@.name == "central")].env[?(@.name == "ROX_OFFLINE_MODE")].value}`,
				"false") == "true",
		},
		"central": map[string]interface{}{
			"disableTelemetry": k8s.evaluateToString(context, "deployment", "central",
				`{.spec.template.spec.containers[?(@.name == "central")].env[?(@.name == "ROX_INIT_TELEMETRY_ENABLED")].value}`, "true") == "false",
			"config":          k8s.evaluateToStringP(context, "configmap", "central-config", `{.data['central-config\.yaml']}`),
			"endpointsConfig": k8s.evaluateToStringP(context, "configmap", "central-endpoints", `{.data['endpoints\.yaml']}`),
			"nodeSelector":    k8s.evaluateToObject(context, "deployment", "central", `{.spec.template.spec.nodeSelector}`, nil),
			"image": map[string]interface{}{
				"registry": extractImageRegistry(k8s.evaluateToString(context, "deployment", "central",
					`{.spec.template.spec.containers[?(@.name == "central")].image}`, ""), "main"),
			},
			"resources": k8s.evaluateToObject(context, "deployment", "central",
				`{.spec.template.spec.containers[?(@.name == "central")].resources}`, nil),
			"persistence": map[string]interface{}{
				"hostPath": k8s.evaluateToStringP(context, "deployment", "central",
					`{.spec.template.spec.volumes[?(@.hostPath)].hostPath.path}`),
				"persistentVolumeClaim": map[string]interface{}{
					"claimName": k8s.evaluateToStringP(context, "deployment", "central",
						`{.spec.template.spec.volumes[?(@.persistentVolumeClaim)].persistentVolumeClaim.claimName}`),
				},
			},
			// Regarding the exposure configuration: Currently we make the assumption that the default port (443) is unchanged.
			// Can be improved to also fetch the port information from the central-loadbalancer service.
			"exposure": map[string]interface{}{
				"loadBalancer": map[string]interface{}{
					"enabled": k8s.evaluateToString(context, "service", "central-loadbalancer", `{.spec.type}`, "") == "LoadBalancer",
				},
				"nodePort": map[string]interface{}{
					"enabled": k8s.evaluateToString(context, "service", "central-loadbalancer", `{.spec.type}`, "") == "NodePort",
				},
			},
		},
		"scanner": scannerConfig,
		"customize": map[string]interface{}{
			"annotations": retrieveCustomAnnotations(k8s.evaluateToObject(context, "deployment", "central",
				`{.metadata.annotations}`, nil)),
			"labels": retrieveCustomLabels(k8s.evaluateToObject(context, "deployment", "central",
				`{.metadata.labels}`, nil)),
			"podLabels": retrieveCustomLabels(k8s.evaluateToObject(context, "deployment", "central",
				`{.spec.template.metadata.labels}`, nil)),
			"podAnnotations": retrieveCustomAnnotations(k8s.evaluateToObject(context, "deployment", "central",
				`{.spec.template.metadata.annotations}`, nil)),
			"envVars": retrieveCustomEnvVars(envVarSliceToObj(k8s.evaluateToSlice(context, "deployment", "central",
				`{.spec.template.spec.containers[?(@.name == "central")].env}`, nil))),
		},
	}

	// Modify YAML marshalling: Recursively remove any keys from objects whose associated values are nil
	// and remove complete objects whose only values are nil.
	mCleaned := mapCopyRemovingNils(m)
	yaml, err := yaml.Marshal(mCleaned)
	if err != nil {
		return errors.Wrap(err, "YAML marshalling failure")
	}

	outputHandle := os.Stdout
	if output != "" && output != "-" {
		outputHandle, err = os.OpenFile(output, os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return errors.Wrapf(err, "open output file %q", output)
		}
	}
	fmt.Fprint(outputHandle, string(yaml))

	if output != "" {
		fmt.Fprintf(os.Stderr, "Helm configuration written to file %q\n", output)
	} else {
		fmt.Fprintln(os.Stderr)
	}

	printWarnings(k8s.getWarnings())

	fmt.Fprintln(os.Stderr,
		`Important: Please verify the correctness of the produced Helm configuration carefully prior to using it.`)

	return nil
}

func retrieveCustomAnnotations(annotations map[string]interface{}) map[string]interface{} {
	return filterMap(annotations, []string{
		"deployment.kubernetes.io/revision",
		"meta.helm.sh/release-name",
		"meta.helm.sh/release-namespace",
		"owner",
		"email",
	})
}

func retrieveCustomLabels(labels map[string]interface{}) map[string]interface{} {
	return filterMap(labels, []string{
		"app",
		"app.kubernets.io/component",
		"app.kubernetes.io/instance",
		"app.kubernetes.io/managed-by",
		"app.kubernetes.io/part-of",
		"app.kubernetes.io/version",
		"app.kubernetes.io/component",
		"app.kubernetes.io/name",
		"helm.sh/chart",
	})
}

func retrieveCustomEnvVars(envVars map[string]interface{}) map[string]interface{} {
	return filterMap(envVars, []string{"ROX_OFFLINE_MODE", "ROX_INIT_TELEMETRY_ENABLED"})
}

// Is there a better way?
func isNil(i interface{}) bool {
	if i == nil {
		return true
	}
	switch reflect.TypeOf(i).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Array, reflect.Chan, reflect.Slice:
		return reflect.ValueOf(i).IsNil()
	}
	return false
}

func printWarnings(warnings []string) {
	if len(warnings) == 0 {
		return
	}
	fmt.Fprintln(os.Stderr, "The following warnings occured:")
	for _, msg := range warnings {
		fmt.Fprintf(os.Stderr, "  WARNING: %s\n", msg)
	}
	fmt.Fprintln(os.Stderr)
}
