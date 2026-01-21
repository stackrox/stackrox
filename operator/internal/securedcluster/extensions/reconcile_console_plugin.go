package extensions

import (
	"context"
	"fmt"

	"dario.cat/mergo"
	"github.com/go-logr/logr"
	consolev1 "github.com/openshift/api/console/v1"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/labels"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	consolePluginName        = "advanced-cluster-security"
	consolePluginDisplayName = "Red Hat Advanced Cluster Security for OpenShift"
	sensorProxyServiceName   = "sensor-proxy"
	sensorProxyPort          = 443
	sensorProxyBasePath      = "/proxy/central/static/ocp-plugin"
)

// ReconcileConsolePluginExtension returns an extension that reconciles the ConsolePlugin CR
// for the OCP Console plugin.
func ReconcileConsolePluginExtension(client ctrlClient.Client, direct ctrlClient.Reader) extensions.ReconcileExtension {
	return wrapExtension(reconcileConsolePlugin, client, direct)
}

func reconcileConsolePlugin(ctx context.Context, sc *platform.SecuredCluster, client ctrlClient.Client, direct ctrlClient.Reader, _ func(statusFunc updateStatusFunc), log logr.Logger) error {
	log = log.WithName("console-plugin")

	if !features.OCPConsoleIntegration.Enabled() {
		return nil
	}

	available, err := isConsolePluginAPIAvailable(ctx, direct, log)
	if err != nil {
		return err
	}
	if !available {
		return nil
	}

	if sc.DeletionTimestamp != nil {
		return deleteConsolePlugin(ctx, client, log)
	}

	return createOrUpdateConsolePlugin(ctx, sc, client, direct, log)
}

func isConsolePluginAPIAvailable(ctx context.Context, reader ctrlClient.Reader, log logr.Logger) (bool, error) {
	list := &consolev1.ConsolePluginList{}
	if err := reader.List(ctx, list); err != nil {
		if meta.IsNoMatchError(err) {
			log.V(1).Info("ConsolePlugin API not present in this cluster")
			return false, nil
		}
		return false, fmt.Errorf("listing ConsolePlugin resources: %w", err)
	}
	return true, nil
}

func deleteConsolePlugin(ctx context.Context, client ctrlClient.Client, log logr.Logger) error {
	plugin := &consolev1.ConsolePlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: consolePluginName,
		},
	}
	if err := client.Delete(ctx, plugin); err != nil {
		if apiErrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("deleting ConsolePlugin: %w", err)
	} else {
		log.Info("Deleted ConsolePlugin", "name", consolePluginName)
	}
	return nil
}

func createOrUpdateConsolePlugin(ctx context.Context, sc *platform.SecuredCluster, client ctrlClient.Client, _ ctrlClient.Reader, log logr.Logger) error {
	desired := buildConsolePlugin(sc)

	plugin := &consolev1.ConsolePlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: consolePluginName,
		},
	}

	result, err := controllerutil.CreateOrUpdate(ctx, client, plugin, func() error {
		if err := mergo.Merge(&plugin.Spec, desired.Spec, mergo.WithOverride); err != nil {
			return fmt.Errorf("merging ConsolePlugin spec: %w", err)
		}

		if plugin.Labels == nil {
			plugin.Labels = make(map[string]string)
		}
		for k, v := range desired.Labels {
			plugin.Labels[k] = v
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("reconciling ConsolePlugin: %w", err)
	}

	if result != controllerutil.OperationResultNone {
		log.Info("Reconciled ConsolePlugin", "name", consolePluginName, "operation", result)
	}
	return nil
}

func buildConsolePlugin(sc *platform.SecuredCluster) *consolev1.ConsolePlugin {
	return &consolev1.ConsolePlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: consolePluginName,
			Labels: map[string]string{
				"app.kubernetes.io/name":    consolePluginName,
				"app.kubernetes.io/part-of": "stackrox-secured-cluster-services",
				labels.ManagedByLabelKey:    labels.ManagedByOperator,
			},
		},
		Spec: consolev1.ConsolePluginSpec{
			DisplayName: consolePluginDisplayName,
			Backend: consolev1.ConsolePluginBackend{
				Type: consolev1.Service,
				Service: &consolev1.ConsolePluginService{
					Name:      sensorProxyServiceName,
					Namespace: sc.Namespace,
					Port:      sensorProxyPort,
					BasePath:  sensorProxyBasePath,
				},
			},
			Proxy: []consolev1.ConsolePluginProxy{
				{
					Alias:         "api-service",
					Authorization: consolev1.UserToken,
					Endpoint: consolev1.ConsolePluginProxyEndpoint{
						Type: consolev1.ProxyTypeService,
						Service: &consolev1.ConsolePluginProxyServiceConfig{
							Name:      sensorProxyServiceName,
							Namespace: sc.Namespace,
							Port:      sensorProxyPort,
						},
					},
				},
			},
		},
	}
}
