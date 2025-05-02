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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type scannerStatusDefaultsReconcilliationTestCase struct {
	Annotations         map[string]string
	ScannerSpec         platform.LocalScannerComponentSpec
	Status              platform.SecuredClusterStatus
	ExpectedAnnotations map[string]string
	ExpectedScannerSpec platform.LocalScannerComponentSpec
}

func TestReconcileScannerFeatureDefaultsExtension(t *testing.T) {
	const annotationKey = defaulting.FeatureDefaultKeySecuredClusterScanner
	cases := map[string]scannerStatusDefaultsReconcilliationTestCase{
		"install: enabled by default": {
			ScannerSpec: platform.LocalScannerComponentSpec{},
			Status:      platform.SecuredClusterStatus{},
			ExpectedScannerSpec: platform.LocalScannerComponentSpec{
				ScannerComponent: &platform.SecuredClusterScannerAutoSense,
			},
			ExpectedAnnotations: map[string]string{
				annotationKey: string(platform.SecuredClusterScannerAutoSense),
			},
		},
		"upgrade: disabled by default": {
			ScannerSpec: platform.LocalScannerComponentSpec{},
			Status:      nonEmptySecuredClusterStatus,
			ExpectedScannerSpec: platform.LocalScannerComponentSpec{
				ScannerComponent: &platform.SecuredClusterScannerDisabled,
			},
			ExpectedAnnotations: map[string]string{
				annotationKey: string(platform.ScannerComponentDisabled),
			},
		},
		"install: enabled explicitly": {
			ScannerSpec: platform.LocalScannerComponentSpec{
				ScannerComponent: &platform.SecuredClusterScannerAutoSense,
			},
			Status: platform.SecuredClusterStatus{},
			ExpectedScannerSpec: platform.LocalScannerComponentSpec{
				ScannerComponent: &platform.SecuredClusterScannerAutoSense,
			},
		},
		"install: disabled explicitly": {
			ScannerSpec: platform.LocalScannerComponentSpec{
				ScannerComponent: &platform.SecuredClusterScannerDisabled,
			},
			Status: platform.SecuredClusterStatus{},
			ExpectedScannerSpec: platform.LocalScannerComponentSpec{
				ScannerComponent: &platform.SecuredClusterScannerDisabled,
			},
		},
		"upgrade: pick up previously persisted default (Enabled)": {
			Status: nonEmptySecuredClusterStatus,
			Annotations: map[string]string{
				annotationKey: string(platform.SecuredClusterScannerAutoSense),
			},
			ExpectedScannerSpec: platform.LocalScannerComponentSpec{
				ScannerComponent: &platform.SecuredClusterScannerAutoSense,
			},
			ExpectedAnnotations: map[string]string{
				annotationKey: string(platform.SecuredClusterScannerAutoSense),
			},
		},
		"upgrade: pick up previously persisted default (Disabled)": {
			Status: nonEmptySecuredClusterStatus,
			Annotations: map[string]string{
				annotationKey: string(platform.ScannerComponentDisabled),
			},
			ExpectedScannerSpec: platform.LocalScannerComponentSpec{
				ScannerComponent: &platform.SecuredClusterScannerDisabled,
			},
			ExpectedAnnotations: map[string]string{
				annotationKey: string(platform.ScannerComponentDisabled),
			},
		},
		"upgrade: ignoring bogus persisted default": {
			Status: nonEmptySecuredClusterStatus,
			Annotations: map[string]string{
				annotationKey: "foo",
			},
			ExpectedScannerSpec: platform.LocalScannerComponentSpec{
				ScannerComponent: &platform.SecuredClusterScannerDisabled,
			},
			ExpectedAnnotations: map[string]string{
				annotationKey: string(platform.ScannerComponentDisabled),
			},
		},
		"previously persisted default is picked up even if status is empty": {
			Annotations: map[string]string{
				annotationKey: string(platform.ScannerComponentDisabled),
			},
			ExpectedScannerSpec: platform.LocalScannerComponentSpec{
				ScannerComponent: &platform.SecuredClusterScannerDisabled,
			},
			ExpectedAnnotations: map[string]string{
				annotationKey: string(platform.ScannerComponentDisabled),
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
					Scanner: c.ScannerSpec.DeepCopy(),
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

			err := reconcileScannerFeatureDefaults(ctx, baseSecuredCluster, client, nil, nil, logr.Discard())
			assert.Nilf(t, err, "reconcileScannerStatusDefaults returned error: %v", err)

			securedCluster := platform.SecuredCluster{}
			err = client.Get(ctx, ctrlClient.ObjectKey{Namespace: testutils.TestNamespace, Name: clusterName}, &securedCluster)
			assert.Nil(t, err, "retrieving SecuredCluster object from fake Kubernetes client")

			// Verify that reconcileScannerFeatureDefaults has modified baseSecuredCluster.Spec as expected.
			assert.Equal(t, baseSecuredCluster.Spec.Scanner, &c.ExpectedScannerSpec,
				"ScannerSpec does not match expected ScannerSpec")

			// Verify that the expected annotations have been persisted via the provided client.
			assert.Equal(t, securedCluster.Annotations, c.ExpectedAnnotations,
				"persisted SecuredCluster annotations do not match expected annotations")

			// Verify that the modified SecuredCluster Spec has not been persisted.
			assert.Equal(t, securedCluster.Spec.Scanner, &c.ScannerSpec, "persisted SecuredCluster spec is unmodified")
		})
	}
}
