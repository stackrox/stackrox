package extensions

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	consolev1 "github.com/openshift/api/console/v1"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/labels"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	consolePluginName        = "advanced-cluster-security"
	consolePluginDisplayName = "Red Hat Advanced Cluster Security for OpenShift"
	consolePluginFinalizer   = "platform.stackrox.io/console-plugin"
	sensorProxyServiceName   = "sensor-proxy"
	sensorProxyPort          = 443
	sensorProxyBasePath      = "/proxy/central/static/ocp-plugin"
)

// ReconcileConsolePluginExtension returns an extension that reconciles the ConsolePlugin CR
// for the OCP Console plugin.
func ReconcileConsolePluginExtension(client ctrlClient.Client, direct ctrlClient.Reader) extensions.ReconcileExtension {
	return func(ctx context.Context, u *unstructured.Unstructured, statusUpdater func(extensions.UpdateStatusFunc), log logr.Logger) error {
		log = log.WithName("console-plugin")

		if u.GroupVersionKind() != platform.SecuredClusterGVK {
			log.Error(errUnexpectedGVK, "unable to reconcile console plugin", "expectedGVK", platform.SecuredClusterGVK, "actualGVK", u.GroupVersionKind())
			return errUnexpectedGVK
		}

		sc := platform.SecuredCluster{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &sc); err != nil {
			return fmt.Errorf("converting object to SecuredCluster: %w", err)
		}

		return reconcileConsolePlugin(ctx, &sc, u, client, direct, log)
	}
}

func reconcileConsolePlugin(ctx context.Context, sc *platform.SecuredCluster, u *unstructured.Unstructured, client ctrlClient.Client, direct ctrlClient.Reader, log logr.Logger) error {
	if !features.OCPConsoleIntegration.Enabled() {
		return nil
	}

	if !isConsolePluginAPIAvailable(ctx, direct, log) {
		return nil
	}

	if sc.DeletionTimestamp != nil {
		return handleConsolePluginDeletion(ctx, u, client, log)
	}

	// Ensure finalizer is present. This is necessary for clean-up on deletion,
	// because the ConsolePlugin is cluster-scoped.
	if !controllerutil.ContainsFinalizer(u, consolePluginFinalizer) {
		controllerutil.AddFinalizer(u, consolePluginFinalizer)
		if err := client.Update(ctx, u); err != nil {
			return fmt.Errorf("adding console plugin finalizer: %w", err)
		}
		log.Info("Added console plugin finalizer to SecuredCluster")
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

func handleConsolePluginDeletion(ctx context.Context, u *unstructured.Unstructured, client ctrlClient.Client, log logr.Logger) error {
	if !controllerutil.ContainsFinalizer(u, consolePluginFinalizer) {
		return nil
	}

	plugin := &consolev1.ConsolePlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: consolePluginName,
		},
	}
	if err := client.Delete(ctx, plugin); err != nil {
		if !apiErrors.IsNotFound(err) {
			return fmt.Errorf("deleting ConsolePlugin: %w", err)
		}
	} else {
		log.Info("Deleted ConsolePlugin", "name", consolePluginName)
	}

	controllerutil.RemoveFinalizer(u, consolePluginFinalizer)
	if err := client.Update(ctx, u); err != nil {
		return fmt.Errorf("removing console plugin finalizer: %w", err)
	}
	log.Info("Removed console plugin finalizer from SecuredCluster")

	return nil
}

func createOrUpdateConsolePlugin(ctx context.Context, sc *platform.SecuredCluster, client ctrlClient.Client, direct ctrlClient.Reader, log logr.Logger) error {
	desired := buildConsolePlugin(sc)

	// Use direct (uncached) reader for cluster-scoped ConsolePlugin
	existing := &consolev1.ConsolePlugin{}
	err := direct.Get(ctx, ctrlClient.ObjectKey{Name: consolePluginName}, existing)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			if err := client.Create(ctx, desired); err != nil {
				return fmt.Errorf("creating ConsolePlugin: %w", err)
			}
			log.Info("Created ConsolePlugin", "name", consolePluginName, "namespace", sc.Namespace)
			return nil
		}
		return fmt.Errorf("getting ConsolePlugin: %w", err)
	}

	existing.Spec = desired.Spec
	existing.Labels = desired.Labels
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
