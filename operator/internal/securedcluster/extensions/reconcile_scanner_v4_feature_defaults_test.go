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
	ScannerV4Spec         platform.LocalScannerV4ComponentSpec
	Status                platform.SecuredClusterStatus
	ExpectedAnnotations   map[string]string
	ExpectedScannerV4Spec platform.LocalScannerV4ComponentSpec
}

var (
	nonEmptyStatus = platform.SecuredClusterStatus{
		DeployedRelease: &platform.StackRoxRelease{
			Version: "some-version-string",
		},
	}
)

func TestReconcileScannerV4FeatureDefaultsExtension(t *testing.T) {
	cases := map[string]scannerV4StatusDefaultsReconcilliationTestCase{
		"install: auto-sense by default": {
			ScannerV4Spec: platform.LocalScannerV4ComponentSpec{},
			Status:        platform.SecuredClusterStatus{},
			ExpectedScannerV4Spec: platform.LocalScannerV4ComponentSpec{
				ScannerComponent: &platform.LocalScannerV4AutoSense,
			},
			ExpectedAnnotations: map[string]string{
				annotationKey: string(platform.LocalScannerV4AutoSense),
			},
		},
		"upgrade: disabled by default": {
			ScannerV4Spec: platform.LocalScannerV4ComponentSpec{},
			Status:        nonEmptyStatus,
			ExpectedScannerV4Spec: platform.LocalScannerV4ComponentSpec{
				ScannerComponent: &platform.LocalScannerV4Disabled,
			},
			ExpectedAnnotations: map[string]string{
				annotationKey: string(platform.ScannerV4ComponentDisabled),
			},
		},
		"install: auto-sense explicitly": {
			ScannerV4Spec: platform.LocalScannerV4ComponentSpec{
				ScannerComponent: &platform.LocalScannerV4AutoSense,
			},
			Status: platform.SecuredClusterStatus{},
			ExpectedScannerV4Spec: platform.LocalScannerV4ComponentSpec{
				ScannerComponent: &platform.LocalScannerV4AutoSense,
			},
		},
		"install: disabled explicitly": {
			ScannerV4Spec: platform.LocalScannerV4ComponentSpec{
				ScannerComponent: &platform.LocalScannerV4Disabled,
			},
			Status: platform.SecuredClusterStatus{},
			ExpectedScannerV4Spec: platform.LocalScannerV4ComponentSpec{
				ScannerComponent: &platform.LocalScannerV4Disabled,
			},
		},
		"upgrade: pick up previously persisted default (AutoSense)": {
			Status: nonEmptyStatus,
			Annotations: map[string]string{
				annotationKey: string(platform.LocalScannerV4AutoSense),
			},
			ExpectedScannerV4Spec: platform.LocalScannerV4ComponentSpec{
				ScannerComponent: &platform.LocalScannerV4AutoSense,
			},
			ExpectedAnnotations: map[string]string{
				annotationKey: string(platform.LocalScannerV4AutoSense),
			},
		},
		"upgrade: pick up previously persisted default (Disabled)": {
			Status: nonEmptyStatus,
			Annotations: map[string]string{
				annotationKey: string(platform.ScannerV4ComponentDisabled),
			},
			ExpectedScannerV4Spec: platform.LocalScannerV4ComponentSpec{
				ScannerComponent: &platform.LocalScannerV4Disabled,
			},
			ExpectedAnnotations: map[string]string{
				annotationKey: string(platform.ScannerV4ComponentDisabled),
			},
		},
		"upgrade: ignoring bogus persisted default": {
			Status: nonEmptyStatus,
			Annotations: map[string]string{
				annotationKey: "foo",
			},
			ExpectedScannerV4Spec: platform.LocalScannerV4ComponentSpec{
				ScannerComponent: &platform.LocalScannerV4Disabled,
			},
			ExpectedAnnotations: map[string]string{
				annotationKey: string(platform.ScannerV4ComponentDisabled),
			},
		},
		"previously persisted default is picked up even if status is empty": {
			Annotations: map[string]string{
				annotationKey: string(platform.ScannerV4ComponentDisabled),
			},
			ExpectedScannerV4Spec: platform.LocalScannerV4ComponentSpec{
				ScannerComponent: &platform.LocalScannerV4Disabled,
			},
			ExpectedAnnotations: map[string]string{
				annotationKey: string(platform.ScannerV4ComponentDisabled),
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			const clusterName = "test-cluster"
			baseSecuredCluster := &platform.SecuredCluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "platform.stackrox.io/v1alpha1",
					Kind:       "SecuredCluster",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:        clusterName,
					Namespace:   testutils.TestNamespace,
					Annotations: make(map[string]string),
				},
				Spec: platform.SecuredClusterSpec{
					ScannerV4: c.ScannerV4Spec.DeepCopy(),
				},
				Status: *c.Status.DeepCopy(),
			}
			for key, val := range c.Annotations {
				baseSecuredCluster.Annotations[key] = val
			}

			ctx := context.Background()
			sch := runtime.NewScheme()
			require.NoError(t, platform.AddToScheme(sch))
			require.NoError(t, scheme.AddToScheme(sch))
			client := fake.NewClientBuilder().
				WithScheme(sch).
				WithObjects(baseSecuredCluster).
				Build()

			err := reconcileScannerV4FeatureDefaults(ctx, baseSecuredCluster, client, nil, nil, logr.Discard())
			assert.Nilf(t, err, "reconcileScannerV4StatusDefaults returned error: %v", err)

			securedCluster := platform.SecuredCluster{}
			err = client.Get(ctx, ctrlClient.ObjectKey{Namespace: testutils.TestNamespace, Name: clusterName}, &securedCluster)
			assert.Nil(t, err, "retrieving SecuredCluster object from fake Kubernetes client")

			// Verify that reconcileScannerV4FeatureDefaults has modified baseSecuredCluster.Spec as expected.
			assert.Equal(t, baseSecuredCluster.Spec.ScannerV4, &c.ExpectedScannerV4Spec,
				"ScannerV4Spec does not match expected ScannerV4Spec")

			// Verify that the expected annotations have been persisted via the provided client.
			assert.Equal(t, securedCluster.Annotations, c.ExpectedAnnotations,
				"persisted SecuredCluster annotations do not match expected annotations")

			// Verify that the modified SecuredCluster Spec has not been persisted.
			assert.Equal(t, securedCluster.Spec.ScannerV4, &c.ScannerV4Spec, "persisted SecuredCluster spec is modified")
		})
	}
}
