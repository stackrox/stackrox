package testutils

import (
	"context"
	"testing"

	consolev1 "github.com/openshift/api/console/v1"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// NewFakeClientBuilder returns a new fake client builder with registered custom resources.
func NewFakeClientBuilder(t *testing.T, objects ...ctrlClient.Object) *fake.ClientBuilder {
	return NewFakeClientBuilderWithConsolePluginListError(t, &meta.NoKindMatchError{
		GroupKind: schema.GroupKind{Group: "console.openshift.io", Kind: "ConsolePlugin"},
	}, objects...)
}

// NewFakeClientBuilderWithConsolePluginListError returns a fake client builder with configurable error
// when listing ConsolePlugins.
// Pass nil to simulate ConsolePlugin API being available.
// Pass a NoKindMatchError to simulate the API not being present (non-OpenShift).
// Pass other errors to simulate API failures (e.g., RBAC errors).
func NewFakeClientBuilderWithConsolePluginListError(t *testing.T, consolePluginListError error, objects ...ctrlClient.Object) *fake.ClientBuilder {
	testutils.MustBeInTest(t)
	scheme := runtime.NewScheme()
	require.NoError(t, platform.AddToScheme(scheme))
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, consolev1.Install(scheme))

	builder := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...)

	if consolePluginListError != nil {
		builder = builder.WithInterceptorFuncs(interceptor.Funcs{
			List: func(ctx context.Context, client ctrlClient.WithWatch, list ctrlClient.ObjectList, opts ...ctrlClient.ListOption) error {
				if _, ok := list.(*consolev1.ConsolePluginList); ok {
					return consolePluginListError
				}
				return client.List(ctx, list, opts...)
			},
		})
	}
	return builder
}
