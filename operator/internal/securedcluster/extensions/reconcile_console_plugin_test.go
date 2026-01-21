package extensions

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	consolev1 "github.com/openshift/api/console/v1"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/utils/testutils"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestReconcileConsolePlugin_NotOnOpenShift(t *testing.T) {
	t.Setenv(features.OCPConsoleIntegration.EnvVar(), "true")

	sc := newTestSecuredCluster()
	scheme := runtime.NewScheme()
	require.NoError(t, platform.AddToScheme(scheme))
	require.NoError(t, consolev1.Install(scheme))

	// Simulate a cluster where the ConsolePlugin CRD doesn't exist.
	noMatchErr := &meta.NoKindMatchError{
		GroupKind: schema.GroupKind{Group: "console.openshift.io", Kind: "ConsolePlugin"},
	}
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(sc).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(ctx context.Context, client ctrlClient.WithWatch, list ctrlClient.ObjectList, opts ...ctrlClient.ListOption) error {
				if _, ok := list.(*consolev1.ConsolePluginList); ok {
					return noMatchErr
				}
				return client.List(ctx, list, opts...)
			},
		}).
		Build()
	u := toUnstructured(t, sc)

	ext := ReconcileConsolePluginExtension(client, client)
	err := ext(context.Background(), u, func(_ extensions.UpdateStatusFunc) {}, logr.Discard())
	require.NoError(t, err, "Extension should succeed hen ConsolePlugin API is not available")

	plugin := &consolev1.ConsolePlugin{}
	err = client.Get(context.Background(), ctrlClient.ObjectKey{Name: consolePluginName}, plugin)
	assert.Error(t, err, "ConsolePlugin should not be created when not on OpenShift")
}

func TestReconcileConsolePlugin_OnOpenShift(t *testing.T) {
	t.Setenv(features.OCPConsoleIntegration.EnvVar(), "true")

	sc := newTestSecuredCluster()
	client := newFakeClientWithConsolePlugin(t, sc)
	u := toUnstructured(t, sc)

	ext := ReconcileConsolePluginExtension(client, client)
	err := ext(context.Background(), u, func(_ extensions.UpdateStatusFunc) {}, logr.Discard())
	require.NoError(t, err)

	plugin := &consolev1.ConsolePlugin{}
	err = client.Get(context.Background(), ctrlClient.ObjectKey{Name: consolePluginName}, plugin)
	require.NoError(t, err)
	assert.Equal(t, testutils.TestNamespace, plugin.Spec.Backend.Service.Namespace)
}

func TestReconcileConsolePlugin_UpdateExisting(t *testing.T) {
	t.Setenv(features.OCPConsoleIntegration.EnvVar(), "true")

	sc := newTestSecuredCluster()

	existingPlugin := &consolev1.ConsolePlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: consolePluginName,
		},
		Spec: consolev1.ConsolePluginSpec{
			DisplayName: "Old Name",
			Backend: consolev1.ConsolePluginBackend{
				Type: consolev1.Service,
				Service: &consolev1.ConsolePluginService{
					Name:      "old-service",
					Namespace: "old-namespace",
					Port:      8080,
				},
			},
		},
	}

	client := newFakeClientWithConsolePlugin(t, sc, existingPlugin)
	u := toUnstructured(t, sc)

	ext := ReconcileConsolePluginExtension(client, client)
	err := ext(context.Background(), u, func(_ extensions.UpdateStatusFunc) {}, logr.Discard())
	require.NoError(t, err)

	plugin := &consolev1.ConsolePlugin{}
	err = client.Get(context.Background(), ctrlClient.ObjectKey{Name: consolePluginName}, plugin)
	require.NoError(t, err)

	assert.Equal(t, consolePluginDisplayName, plugin.Spec.DisplayName)
	assert.Equal(t, sensorProxyServiceName, plugin.Spec.Backend.Service.Name)
	assert.Equal(t, testutils.TestNamespace, plugin.Spec.Backend.Service.Namespace)

	assert.Equal(t, consolePluginName, plugin.Labels["app.kubernetes.io/name"])
	assert.Equal(t, "stackrox-secured-cluster-services", plugin.Labels["app.kubernetes.io/part-of"])
	assert.Equal(t, labels.ManagedByOperator, plugin.Labels[labels.ManagedByLabelKey])
}

func TestReconcileConsolePlugin_Deletion(t *testing.T) {
	t.Setenv(features.OCPConsoleIntegration.EnvVar(), "true")

	existingPlugin := &consolev1.ConsolePlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: consolePluginName,
		},
		Spec: consolev1.ConsolePluginSpec{
			DisplayName: consolePluginDisplayName,
		},
	}

	sc := newTestSecuredCluster()
	now := metav1.Now()
	sc.DeletionTimestamp = &now
	sc.Finalizers = []string{"test-finalizer"}

	client := newFakeClientWithConsolePlugin(t, sc, existingPlugin)
	u := toUnstructured(t, sc)

	ext := ReconcileConsolePluginExtension(client, client)
	err := ext(context.Background(), u, func(_ extensions.UpdateStatusFunc) {}, logr.Discard())
	require.NoError(t, err)

	plugin := &consolev1.ConsolePlugin{}
	err = client.Get(context.Background(), ctrlClient.ObjectKey{Name: consolePluginName}, plugin)
	assert.Error(t, err, "ConsolePlugin should be deleted")
}

func TestReconcileConsolePlugin_DeletionWithoutPlugin(t *testing.T) {
	t.Setenv(features.OCPConsoleIntegration.EnvVar(), "true")

	sc := newTestSecuredCluster()
	now := metav1.Now()
	sc.DeletionTimestamp = &now
	sc.Finalizers = []string{"test-finalizer"}

	client := newFakeClientWithConsolePlugin(t, sc)
	u := toUnstructured(t, sc)

	ext := ReconcileConsolePluginExtension(client, client)
	err := ext(context.Background(), u, func(_ extensions.UpdateStatusFunc) {}, logr.Discard())
	require.NoError(t, err)
}

func newTestSecuredCluster() *platform.SecuredCluster {
	clusterName := "test-cluster"
	return &platform.SecuredCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "platform.stackrox.io/v1alpha1",
			Kind:       "SecuredCluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: testutils.TestNamespace,
		},
		Spec: platform.SecuredClusterSpec{
			ClusterName: &clusterName,
		},
	}
}

func newFakeClientWithConsolePlugin(t *testing.T, objects ...ctrlClient.Object) ctrlClient.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, platform.AddToScheme(scheme))
	require.NoError(t, consolev1.Install(scheme))

	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		Build()
}
