package extensions

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/utils/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type scannerV4StatusDefaultsReconcilliationTestCase struct {
	Annotations           map[string]string
	ScannerV4Spec         platform.ScannerV4Spec
	Status                platform.CentralStatus
	ExpectedAnnotations   map[string]string
	ExpectedScannerV4Spec platform.ScannerV4Spec
}

var (
	nonEmptyStatus = platform.CentralStatus{
		DeployedRelease: &platform.StackRoxRelease{
			Name: "release-name",
		},
	}
)

func TestReconcileScannerV4FeatureDefaultsExtension(t *testing.T) {
	cases := map[string]scannerV4StatusDefaultsReconcilliationTestCase{
		"install: enabled by default": {
			ScannerV4Spec: platform.ScannerV4Spec{},
			Status:        platform.CentralStatus{},
			ExpectedScannerV4Spec: platform.ScannerV4Spec{
				ScannerComponent: &platform.ScannerV4Enabled,
			},
			ExpectedAnnotations: map[string]string{
				annotationKey: string(platform.ScannerV4ComponentEnabled),
			},
		},
		"upgrade: disabled by default": {
			ScannerV4Spec: platform.ScannerV4Spec{},
			Status:        nonEmptyStatus,
			ExpectedScannerV4Spec: platform.ScannerV4Spec{
				ScannerComponent: &platform.ScannerV4Disabled,
			},
			ExpectedAnnotations: map[string]string{
				annotationKey: string(platform.ScannerV4ComponentDisabled),
			},
		},
		"install: enabled explicitly": {
			ScannerV4Spec: platform.ScannerV4Spec{
				ScannerComponent: &platform.ScannerV4Enabled,
			},
			Status: platform.CentralStatus{},
			ExpectedScannerV4Spec: platform.ScannerV4Spec{
				ScannerComponent: &platform.ScannerV4Enabled,
			},
		},
		"install: disabled explicitly": {
			ScannerV4Spec: platform.ScannerV4Spec{
				ScannerComponent: &platform.ScannerV4Disabled,
			},
			Status: platform.CentralStatus{},
			ExpectedScannerV4Spec: platform.ScannerV4Spec{
				ScannerComponent: &platform.ScannerV4Disabled,
			},
		},
		"upgrade: pick up previously persisted default (Enabled)": {
			Status: nonEmptyStatus,
			Annotations: map[string]string{
				annotationKey: string(platform.ScannerV4ComponentEnabled),
			},
			ExpectedScannerV4Spec: platform.ScannerV4Spec{
				ScannerComponent: &platform.ScannerV4Enabled,
			},
			ExpectedAnnotations: map[string]string{
				annotationKey: string(platform.ScannerV4ComponentEnabled),
			},
		},
		"upgrade: pick up previously persisted default (Disabled)": {
			Status: nonEmptyStatus,
			Annotations: map[string]string{
				annotationKey: string(platform.ScannerV4ComponentDisabled),
			},
			ExpectedScannerV4Spec: platform.ScannerV4Spec{
				ScannerComponent: &platform.ScannerV4Disabled,
			},
			ExpectedAnnotations: map[string]string{
				annotationKey: string(platform.ScannerV4ComponentDisabled),
			},
		},
		"upgrade: pick up previously persisted default (else)": {
			Status: nonEmptyStatus,
			Annotations: map[string]string{
				annotationKey: "foo",
			},
			ExpectedScannerV4Spec: platform.ScannerV4Spec{
				ScannerComponent: &platform.ScannerV4Disabled,
			},
			ExpectedAnnotations: map[string]string{
				annotationKey: "foo",
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			const centralName = "test-central"
			baseCentral := &platform.Central{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "platform.stackrox.io/v1alpha1",
					Kind:       "Central",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:        centralName,
					Namespace:   testutils.TestNamespace,
					Annotations: make(map[string]string),
				},
				Spec: platform.CentralSpec{
					ScannerV4: c.ScannerV4Spec.DeepCopy(),
				},
				Status: *c.Status.DeepCopy(),
			}
			for key, val := range c.Annotations {
				baseCentral.Annotations[key] = val
			}

			ctx := context.Background()
			sch := runtime.NewScheme()
			require.NoError(t, platform.AddToScheme(sch))
			require.NoError(t, scheme.AddToScheme(sch))
			client := fake.NewClientBuilder().
				WithScheme(sch).
				WithObjects(baseCentral).
				Build()

			err := reconcileScannerV4FeatureDefaults(ctx, baseCentral, client, nil, nil, logr.Discard())
			assert.Nilf(t, err, "reconcileScannerV4StatusDefaults returned error: %v", err)

			central := platform.Central{}
			err = client.Get(ctx, ctrlClient.ObjectKey{Namespace: testutils.TestNamespace, Name: centralName}, &central)
			assert.Nil(t, err, "retrieving Central object from fake Kubernetes client")

			// Verify that reconcileScannerV4FeatureDefaults has modified baseCentral.Spec as expected.
			assert.Equal(t, baseCentral.Spec.ScannerV4, &c.ExpectedScannerV4Spec,
				"central annotations to not match expected annotations")

			// Verify that the expected annotations have been persisted via the provided client.
			assert.Equal(t, central.Annotations, c.ExpectedAnnotations,
				"persisted central annotations do not match expected annotations")

			// Verify that the modified Central Spec has not been persisted.
			assert.Equal(t, central.Spec.ScannerV4, &c.ScannerV4Spec, "persisted central spec is unmodified")
		})
	}
}
