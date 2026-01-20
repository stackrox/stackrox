package extensions

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	consolev1 "github.com/openshift/api/console/v1"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/utils"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/labels"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
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

	if !features.OCPConsoleIntegration.Enabled() || !isConsolePluginAPIAvailable(ctx, direct, log) {
		return nil
	}

	if sc.DeletionTimestamp != nil {
		return deleteConsolePlugin(ctx, client, log)
	}

	return createOrUpdateConsolePlugin(ctx, sc, client, direct, log)
}

func isConsolePluginAPIAvailable(ctx context.Context, reader ctrlClient.Reader, log logr.Logger) bool {
	list := &consolev1.ConsolePluginList{}
	if err := reader.List(ctx, list); err != nil {
		if meta.IsNoMatchError(err) {
			log.V(1).Info("ConsolePlugin API not present in this cluster", "error", err.Error())
		} else {
			log.Error(err, "Failed to list ConsolePlugin resources; treating ConsolePlugin API as unavailable")
		}
		return false
	}
	return true
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

func createOrUpdateConsolePlugin(ctx context.Context, sc *platform.SecuredCluster, client ctrlClient.Client, direct ctrlClient.Reader, log logr.Logger) error {
	desired := buildConsolePlugin(sc)

	existing := &consolev1.ConsolePlugin{}
	err := utils.GetWithFallbackToUncached(ctx, client, direct, ctrlClient.ObjectKey{Name: consolePluginName}, existing)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			if err := client.Create(ctx, desired); err != nil {
				return fmt.Errorf("creating ConsolePlugin: %w", err)
			}
			log.Info("Created ConsolePlugin", "name", consolePluginName)
			return nil
		}
		return fmt.Errorf("getting ConsolePlugin: %w", err)
	}

	existing.Spec = desired.Spec
	if existing.Labels == nil {
		existing.Labels = make(map[string]string)
	}
	for k, v := range desired.Labels {
		existing.Labels[k] = v
	}

	if err := client.Update(ctx, existing); err != nil {
		return fmt.Errorf("updating ConsolePlugin: %w", err)
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
