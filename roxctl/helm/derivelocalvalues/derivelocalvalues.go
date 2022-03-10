package derivelocalvalues

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/roxctl/helm/internal/common"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chartutil"
)

var (
	supportedCharts = []string{common.ChartCentralServices}
)

func deriveLocalValuesForChart(namespace, chartName, input, output string, useDirectory bool) error {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()
	switch chartName {
	case common.ChartCentralServices:
		err = deriveLocalValuesForCentralServices(ctx, namespace, input, output, useDirectory)
	default:
		fmt.Fprintf(os.Stderr, "Deriving local values for chart %q is currently unsupported.\n", chartName)
		fmt.Fprintf(os.Stderr, "Supported charts: %s\n", strings.Join(supportedCharts, ", "))
		err = errors.Errorf("unsupported chart %q", chartName)
	}

	return err
}

// Remove nils from the given map, serialize it as YAML and write it to the output stream.
func writeYamlToStream(values map[string]interface{}, outputHandle *os.File) error {
	yaml, err := yaml.Marshal(values)
	if err != nil {
		return errors.Wrap(err, "YAML marshalling")
	}

	_, err = outputHandle.Write(yaml)
	if err != nil {
		return errors.Wrap(err, "writing YAML configuration")
	}

	return nil
}

func writeYamlToFile(values map[string]interface{}, path string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return errors.Wrapf(err, "opening file %q", path)
	}
	fileToClose := file

	defer func() {
		if fileToClose != nil {
			_ = fileToClose.Close()
		}
	}()

	err = writeYamlToStream(values, file)
	if err != nil {
		return errors.Wrapf(err, "writing YAML to file %q", path)
	}

	fileToClose = nil
	if err := file.Close(); err != nil {
		return errors.Wrapf(err, "closing file %q", path)
	}

	return nil
}

func writeValuesToOutput(publicValues, privateValues map[string]interface{}, output string, useDirectory bool) error {
	var err error

	if useDirectory {
		// Write to file(s) in output directory.

		err = os.MkdirAll(output, 0700)
		if err != nil {
			return errors.Wrapf(err, "creating output directory %q", output)
		}

		publicErr := writeYamlToFile(publicValues, filepath.Join(output, "values-public.yaml"))
		if publicErr != nil {
			err = multierror.Append(err, publicErr)
		}
		privateErr := writeYamlToFile(privateValues, filepath.Join(output, "values-private.yaml"))
		if privateErr != nil {
			err = multierror.Append(err, privateErr)
		}
	} else {
		// write everything to a single file or stdout.
		allValues := chartutil.CoalesceTables(publicValues, privateValues)

		if output == "" {
			err = writeYamlToStream(allValues, os.Stdout)
			// Add a newline to delimit the YAML from other output for the user.
			fmt.Fprintln(os.Stderr)
		} else {
			err = writeYamlToFile(allValues, output)
		}
	}

	if err != nil {
		return err
	}

	return nil
}

// Implementation for command `helm derive-local-values`.
func deriveLocalValuesForCentralServices(ctx context.Context, namespace, input, output string, useDirectory bool) error {
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
		k8sLocal, err := newLocalK8sObjectDescriptionFromPath(input)
		if err != nil {
			return errors.Wrapf(err, "retrieving Kubernetes resource definitions from %q", input)
		}
		k8s = newK8sObjectDescription(k8sLocal)
	}

	publicValues, privateValues, err := helmValuesForCentralServices(ctx, namespace, k8s)
	if err != nil {
		return errors.Wrap(err, "deriving local values")
	}

	err = writeValuesToOutput(publicValues, privateValues, output, useDirectory)
	if err != nil {
		return errors.Wrap(err, "writing configuration")
	}

	printWarnings(k8s.getWarnings())

	fmt.Fprintln(os.Stderr,
		`Important: Please verify the correctness of the produced Helm configuration carefully prior to using it.`)

	return nil
}

// Implementation for command `helm derive-local-values`.
func helmValuesForCentralServices(ctx context.Context, namespace string, k8s k8sObjectDescription) (map[string]interface{}, map[string]interface{}, error) {
	var err error

	publicValues, publicErr := derivePublicLocalValuesForCentralServices(ctx, namespace, k8s)
	if publicErr != nil {
		err = multierror.Append(err, publicErr)
	}
	privateValues, privateErr := derivePrivateLocalValuesForCentralServices(ctx, namespace, k8s)
	if privateErr != nil {
		err = multierror.Append(err, privateErr)
	}

	// Normalize value maps:
	// - recursively remove any keys from objects whose associated values are nil,
	// - remove complete objects whose only values are nil,
	// - replace string pointers with strings.
	publicValuesCleaned := maputil.NormalizeGenericMap(publicValues)
	privateValuesCleaned := maputil.NormalizeGenericMap(privateValues)

	return publicValuesCleaned, privateValuesCleaned, errors.Wrap(err, "could not derive local values")

}

// Implementation for command `helm derive-local-values`.
func derivePrivateLocalValuesForCentralServices(ctx context.Context, namespace string, k8s k8sObjectDescription) (map[string]interface{}, error) {
	m := map[string]interface{}{
		"licenseKey": k8s.lookupSecretStringP(ctx, "central-license", "license.lic"),
		"env": map[string]interface{}{
			"proxyConfig": k8s.lookupSecretStringP(ctx, "proxy-config", "config.yaml"),
		},
		"ca": map[string]interface{}{
			"cert": k8s.lookupSecretStringP(ctx, "central-tls", "ca.pem"),
			"key":  k8s.lookupSecretStringP(ctx, "central-tls", "ca-key.pem"),
		},
		"central": map[string]interface{}{
			"jwtSigner": map[string]interface{}{
				"key": k8s.lookupSecretStringP(ctx, "central-tls", "jwt-key.pem"),
			},
			"serviceTLS": map[string]interface{}{
				"cert": k8s.lookupSecretStringP(ctx, "central-tls", "cert.pem"),
				"key":  k8s.lookupSecretStringP(ctx, "central-tls", "key.pem"),
			},
			"defaultTLS": map[string]interface{}{
				"cert": k8s.lookupSecretStringP(ctx, "central-default-tls-cert", "tls.crt"),
				"key":  k8s.lookupSecretStringP(ctx, "central-default-tls-cert", "tls.key"),
			},
			"adminPassword": map[string]interface{}{
				"htpasswd": k8s.lookupSecretStringP(ctx, "central-htpasswd", "htpasswd"),
			},
		},
		"scanner": map[string]interface{}{
			"dbPassword": map[string]interface{}{
				"value": k8s.lookupSecretStringP(ctx, "scanner-db-password", "password"),
			},
			"serviceTLS": map[string]interface{}{
				"cert": k8s.lookupSecretStringP(ctx, "scanner-tls", "cert.pem"),
				"key":  k8s.lookupSecretStringP(ctx, "scanner-tls", "key.pem"),
			},
			"dbServiceTLS": map[string]interface{}{
				"cert": k8s.lookupSecretStringP(ctx, "scanner-db-tls", "cert.pem"),
				"key":  k8s.lookupSecretStringP(ctx, "scanner-db-tls", "key.pem"),
			},
		},
	}

	return m, nil
}

// Implementation for command `helm derive-local-values`.
func derivePublicLocalValuesForCentralServices(ctx context.Context, namespace string, k8s k8sObjectDescription) (map[string]interface{}, error) {

	// Note regarding custom metadata (annotations, labels and env vars): We make it easy for us:
	// we simply retrieve the metadata from the central deployment and assume that any custom metadata
	// on that resource is to be used globally for all StackRox resources.

	var scannerConfig map[string]interface{}
	if k8s.Exists(ctx, "deployment", "scanner") {
		scannerConfig = map[string]interface{}{
			"replicas": k8s.evaluateToInt64(ctx, "deployment", "scanner", `{.spec.replicas}`, 3),
			"autoscaling": k8s.evaluateToSubObject(ctx, "hpa", "scanner", `{.spec}`, []string{"minReplicas", "maxReplicas"},
				map[string]interface{}{"disable": true}),
			"resources": k8s.evaluateToObject(ctx, "deployment", "scanner",
				`{.spec.template.spec.containers[?(@.name == "scanner")].resources}`, nil),
			"image": map[string]interface{}{
				"registry": extractImageRegistry(k8s.evaluateToString(ctx, "deployment", "scanner",
					`{.spec.template.spec.containers[?(@.name == "scanner")].image}`, ""), "scanner"),
			},
			"dbImage": map[string]interface{}{
				"registry": extractImageRegistry(k8s.evaluateToString(ctx, "deployment", "scanner-db",
					`{.spec.template.spec.containers[?(@.name == "db")].image}`, ""), "scanner-db"),
			},
			"dbResources": k8s.evaluateToObject(ctx, "deployment", "scanner-db",
				`{.spec.template.spec.containers[?(@.name == "db")].resources}`, nil),
		}
	} else {
		scannerConfig = map[string]interface{}{
			"disable": true,
		}
	}

	m := map[string]interface{}{
		// "image": We do not specify a global registry,
		// instead we only specify central- and scanner-specific registries.
		"env": map[string]interface{}{
			"offlineMode": k8s.evaluateToString(ctx, "deployment", "central",
				`{.spec.template.spec.containers[?(@.name == "central")].env[?(@.name == "ROX_OFFLINE_MODE")].value}`,
				"false") == "true",
		},
		"central": map[string]interface{}{
			"disableTelemetry": k8s.evaluateToString(ctx, "deployment", "central",
				`{.spec.template.spec.containers[?(@.name == "central")].env[?(@.name == "ROX_INIT_TELEMETRY_ENABLED")].value}`, "true") == "false",
			"config":          k8s.evaluateToStringP(ctx, "configmap", "central-config", `{.data['central-config\.yaml']}`),
			"endpointsConfig": k8s.evaluateToStringP(ctx, "configmap", "central-endpoints", `{.data['endpoints\.yaml']}`),
			"nodeSelector":    k8s.evaluateToObject(ctx, "deployment", "central", `{.spec.template.spec.nodeSelector}`, nil),
			"image": map[string]interface{}{
				"registry": extractImageRegistry(k8s.evaluateToString(ctx, "deployment", "central",
					`{.spec.template.spec.containers[?(@.name == "central")].image}`, ""), "main"),
			},
			"resources": k8s.evaluateToObject(ctx, "deployment", "central",
				`{.spec.template.spec.containers[?(@.name == "central")].resources}`, nil),
			"persistence": map[string]interface{}{
				"hostPath": k8s.evaluateToStringP(ctx, "deployment", "central",
					`{.spec.template.spec.volumes[?(@.hostPath)].hostPath.path}`),
				"persistentVolumeClaim": map[string]interface{}{
					"claimName": k8s.evaluateToStringP(ctx, "deployment", "central",
						`{.spec.template.spec.volumes[?(@.persistentVolumeClaim)].persistentVolumeClaim.claimName}`),
				},
			},
			// Regarding the exposure configuration: Currently we make the assumption that the default port (443) is unchanged.
			// Can be improved to also fetch the port information from the central-loadbalancer service.
			"exposure": map[string]interface{}{
				"loadBalancer": map[string]interface{}{
					"enabled": k8s.evaluateToString(ctx, "service", "central-loadbalancer", `{.spec.type}`, "") == "LoadBalancer",
				},
				"nodePort": map[string]interface{}{
					"enabled": k8s.evaluateToString(ctx, "service", "central-loadbalancer", `{.spec.type}`, "") == "NodePort",
				},
			},
			"enableCentralDB": k8s.evaluateToString(ctx, "service", "central-db", `{.spec.type}`, "") != "",
		},
		"scanner": scannerConfig,
		"customize": map[string]interface{}{
			"annotations": retrieveCustomAnnotations(k8s.evaluateToObject(ctx, "deployment", "central",
				`{.metadata.annotations}`, nil)),
			"labels": retrieveCustomLabels(k8s.evaluateToObject(ctx, "deployment", "central",
				`{.metadata.labels}`, nil)),
			"podLabels": retrieveCustomLabels(k8s.evaluateToObject(ctx, "deployment", "central",
				`{.spec.template.metadata.labels}`, nil)),
			"podAnnotations": retrieveCustomAnnotations(k8s.evaluateToObject(ctx, "deployment", "central",
				`{.spec.template.metadata.annotations}`, nil)),
			"envVars": retrieveCustomEnvVars(envVarSliceToObj(k8s.evaluateToSlice(ctx, "deployment", "central",
				`{.spec.template.spec.containers[?(@.name == "central")].env}`, nil))),
		},
	}
	return m, nil
}

func retrieveCustomAnnotations(annotations map[string]interface{}) map[string]interface{} {
	return filterMap(annotations, []string{
		"deployment.kubernetes.io/revision",
		"meta.helm.sh/release-name",
		"meta.helm.sh/release-namespace",
		"owner",
		"email",
		"traffic.sidecar.istio.io/excludeInboundPorts",
	})
}

func retrieveCustomLabels(labels map[string]interface{}) map[string]interface{} {
	return filterMap(labels, []string{
		"app",
		"app.kubernets.io/component", // typo that existed in old versions
		"app.kubernetes.io/component",
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
