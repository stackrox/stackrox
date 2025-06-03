package extensions

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/common/defaulting"
	"github.com/stackrox/rox/operator/internal/utils/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type scannerV4StatusDefaultsReconcilliationTestCase struct {
	Annotations         map[string]string
	Spec                platform.CentralSpec
	Status              platform.CentralStatus
	ExpectedAnnotations map[string]string
	ExpectedDefaults    platform.CentralSpec
}

var (
	nonEmptyStatus = platform.CentralStatus{
		DeployedRelease: &platform.StackRoxRelease{
			Version: "some-version-string",
		},
	}
)

func TestReconcileScannerV4FeatureDefaultsExtension(t *testing.T) {
	cases := map[string]scannerV4StatusDefaultsReconcilliationTestCase{
		"install: scanner V4 enabled by default": {
			Spec:   platform.CentralSpec{},
			Status: platform.CentralStatus{},
			ExpectedDefaults: platform.CentralSpec{
				ScannerV4: &platform.ScannerV4Spec{
					ScannerComponent: &platform.ScannerV4Enabled,
				},
			},
			ExpectedAnnotations: map[string]string{
				defaulting.FeatureDefaultKeyScannerV4: string(platform.ScannerV4ComponentEnabled),
			},
		},
		"upgrade: disabled by default": {
			Spec:   platform.CentralSpec{},
			Status: nonEmptyStatus,
			ExpectedDefaults: platform.CentralSpec{
				ScannerV4: &platform.ScannerV4Spec{
					ScannerComponent: &platform.ScannerV4Disabled,
				},
			},
			ExpectedAnnotations: map[string]string{
				defaulting.FeatureDefaultKeyScannerV4: string(platform.ScannerV4ComponentDisabled),
			},
		},
		"install: enabled explicitly": {
			Spec: platform.CentralSpec{
				ScannerV4: &platform.ScannerV4Spec{
					ScannerComponent: &platform.ScannerV4Enabled,
				},
			},
			Status:           platform.CentralStatus{},
			ExpectedDefaults: platform.CentralSpec{},
		},
		"install: disabled explicitly": {
			Spec: platform.CentralSpec{
				ScannerV4: &platform.ScannerV4Spec{
					ScannerComponent: &platform.ScannerV4Disabled,
				},
			},
			Status:           platform.CentralStatus{},
			ExpectedDefaults: platform.CentralSpec{},
		},
		"upgrade: pick up previously persisted default (Enabled)": {
			Status: nonEmptyStatus,
			Annotations: map[string]string{
				defaulting.FeatureDefaultKeyScannerV4: string(platform.ScannerV4ComponentEnabled),
			},
			ExpectedDefaults: platform.CentralSpec{
				ScannerV4: &platform.ScannerV4Spec{
					ScannerComponent: &platform.ScannerV4Enabled,
				},
			},
			ExpectedAnnotations: map[string]string{
				defaulting.FeatureDefaultKeyScannerV4: string(platform.ScannerV4ComponentEnabled),
			},
		},
		"upgrade: pick up previously persisted default (Disabled)": {
			Status: nonEmptyStatus,
			Annotations: map[string]string{
				defaulting.FeatureDefaultKeyScannerV4: string(platform.ScannerV4ComponentDisabled),
			},
			ExpectedDefaults: platform.CentralSpec{
				ScannerV4: &platform.ScannerV4Spec{
					ScannerComponent: &platform.ScannerV4Disabled,
				},
			},
			ExpectedAnnotations: map[string]string{
				defaulting.FeatureDefaultKeyScannerV4: string(platform.ScannerV4ComponentDisabled),
			},
		},
		"upgrade: ignoring bogus persisted default": {
			Status: nonEmptyStatus,
			Annotations: map[string]string{
				defaulting.FeatureDefaultKeyScannerV4: "foo",
			},
			ExpectedDefaults: platform.CentralSpec{
				ScannerV4: &platform.ScannerV4Spec{
					ScannerComponent: &platform.ScannerV4Disabled,
				},
			},
			ExpectedAnnotations: map[string]string{
				defaulting.FeatureDefaultKeyScannerV4: string(platform.ScannerV4ComponentDisabled),
			},
		},
		"previously persisted default is picked up even if status is empty": {
			Annotations: map[string]string{
				defaulting.FeatureDefaultKeyScannerV4: string(platform.ScannerV4ComponentDisabled),
			},
			ExpectedDefaults: platform.CentralSpec{
				ScannerV4: &platform.ScannerV4Spec{
					ScannerComponent: &platform.ScannerV4Disabled,
				},
			},
			ExpectedAnnotations: map[string]string{
				defaulting.FeatureDefaultKeyScannerV4: string(platform.ScannerV4ComponentDisabled),
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			const centralName = "test-central"
			central := &platform.Central{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "platform.stackrox.io/v1alpha1",
					Kind:       "Central",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:        centralName,
					Namespace:   testutils.TestNamespace,
					Annotations: make(map[string]string),
				},
				Spec:   c.Spec,
				Status: *c.Status.DeepCopy(),
			}
			for key, val := range c.Annotations {
				central.Annotations[key] = val
			}

			ctx := context.Background()
			sch := runtime.NewScheme()
			require.NoError(t, platform.AddToScheme(sch))
			require.NoError(t, scheme.AddToScheme(sch))
			client := fake.NewClientBuilder().
				WithScheme(sch).
				WithObjects(central).
				Build()
			unstructuredCentral := centralToUnstructured(t, central)

			err := reconcileFeatureDefaults(ctx, client, unstructuredCentral, logr.Discard())
			assert.Nil(t, err, "reconcileScannerV4StatusDefaults returned error")

			centralFetched := platform.Central{}
			err = client.Get(ctx, ctrlClient.ObjectKey{Namespace: testutils.TestNamespace, Name: centralName}, &centralFetched)
			assert.Nil(t, err, "retrieving Central object from fake Kubernetes client")

			centralDefaults := extractCentralDefaults(t, unstructuredCentral)

			// Verify that reconcileScannerV4FeatureDefaults has modified the Defaults as expected.
			assert.Equal(t, centralDefaults, &c.ExpectedDefaults, "Central Defaults do not match expected Defaults")

			// Verify that the expected annotations have been persisted via the provided client.
			assert.Equal(t, centralFetched.Annotations, c.ExpectedAnnotations, "persisted central annotations do not match expected annotations")

			// Verify that the Central Spec on the Cluster is unmodified.
			assert.Equal(t, centralFetched.Spec, c.Spec, "persisted central spec is modified")
		})
	}
}

func centralToUnstructured(t *testing.T, central *platform.Central) *unstructured.Unstructured {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(central)
	assert.NoError(t, err)
	return &unstructured.Unstructured{Object: obj}
}

func extractCentralDefaults(t *testing.T, u *unstructured.Unstructured) *platform.CentralSpec {
	defaults := platform.CentralSpec{}
	unstructuredCentralDefaults, ok := u.Object["defaults"].(map[string]interface{})
	if ok {
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredCentralDefaults, &defaults)
		assert.Nil(t, err, "failed to extract Central Defaults from unstructured object")
	}
	return &defaults
}
