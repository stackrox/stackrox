package extensions

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/common/defaulting"
	"github.com/stackrox/rox/operator/internal/securedcluster/values/defaults"

	"github.com/stackrox/rox/operator/internal/utils/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type scannerV4DefaultingTestCase struct {
	Annotations         map[string]string
	Spec                platform.SecuredClusterSpec
	Status              platform.SecuredClusterStatus
	ExpectedAnnotations map[string]string
	ExpectedDefaults    *platform.LocalScannerV4ComponentSpec
}

type admissionControllerDefaultingTestCase struct {
	Annotations         map[string]string
	Spec                platform.SecuredClusterSpec
	Status              platform.SecuredClusterStatus
	ExpectedAnnotations map[string]string
	ExpectedDefaults    *platform.AdmissionControlComponentSpec
}

var (
	nonEmptyStatus = platform.SecuredClusterStatus{
		DeployedRelease: &platform.StackRoxRelease{
			Version: "some-version-string",
		},
	}
)

func TestReconcileAdmissionControllerDefaulting(t *testing.T) {
	t.Setenv("ROX_ADMISSION_CONTROLLER_CONFIG", "true")
	cases := map[string]admissionControllerDefaultingTestCase{
		"install: empty spec": {
			Spec:   platform.SecuredClusterSpec{},
			Status: platform.SecuredClusterStatus{},
			ExpectedDefaults: &platform.AdmissionControlComponentSpec{
				Bypass:        ptr.To(platform.BypassBreakGlassAnnotation),
				FailurePolicy: ptr.To(platform.FailurePolicyIgnore),
				Replicas:      ptr.To(int32(3)),
				Enforce:       ptr.To(true),
			},
			ExpectedAnnotations: map[string]string{
				defaults.FeatureDefaultKeyAdmissionControllerEnforce: "true",
			},
		},
		"upgrade: annotation true is picked up": {
			Spec:   platform.SecuredClusterSpec{},
			Status: nonEmptyStatus,
			Annotations: map[string]string{
				defaults.FeatureDefaultKeyAdmissionControllerEnforce: "true",
			},
			ExpectedDefaults: &platform.AdmissionControlComponentSpec{
				Bypass:        ptr.To(platform.BypassBreakGlassAnnotation),
				FailurePolicy: ptr.To(platform.FailurePolicyIgnore),
				Replicas:      ptr.To(int32(3)),
				Enforce:       ptr.To(true),
			},
			ExpectedAnnotations: map[string]string{
				defaults.FeatureDefaultKeyAdmissionControllerEnforce: "true",
			},
		},
		"upgrade: annotation false is picked up": {
			Spec:   platform.SecuredClusterSpec{},
			Status: nonEmptyStatus,
			Annotations: map[string]string{
				defaults.FeatureDefaultKeyAdmissionControllerEnforce: "false",
			},
			ExpectedDefaults: &platform.AdmissionControlComponentSpec{
				Bypass:        ptr.To(platform.BypassBreakGlassAnnotation),
				FailurePolicy: ptr.To(platform.FailurePolicyIgnore),
				Replicas:      ptr.To(int32(3)),
				Enforce:       ptr.To(false),
			},
			ExpectedAnnotations: map[string]string{
				defaults.FeatureDefaultKeyAdmissionControllerEnforce: "false",
			},
		},
		"upgrade: enforce disabled if listenOnCreates & listenOnUpdates disabled": {
			Spec: platform.SecuredClusterSpec{
				AdmissionControl: &platform.AdmissionControlComponentSpec{
					ListenOnCreates: ptr.To(false),
					ListenOnUpdates: ptr.To(false),
				},
			},
			Status: nonEmptyStatus,
			ExpectedDefaults: &platform.AdmissionControlComponentSpec{
				Bypass:        ptr.To(platform.BypassBreakGlassAnnotation),
				FailurePolicy: ptr.To(platform.FailurePolicyIgnore),
				Replicas:      ptr.To(int32(3)),
				Enforce:       ptr.To(false),
			},
			ExpectedAnnotations: map[string]string{
				defaults.FeatureDefaultKeyAdmissionControllerEnforce: "false",
			},
		},
		"upgrade: enforce enabled if listenOnCreates enabled": {
			Spec: platform.SecuredClusterSpec{
				AdmissionControl: &platform.AdmissionControlComponentSpec{
					ListenOnCreates: ptr.To(true),
					ListenOnUpdates: ptr.To(false),
				},
			},
			Status: nonEmptyStatus,
			ExpectedDefaults: &platform.AdmissionControlComponentSpec{
				Bypass:        ptr.To(platform.BypassBreakGlassAnnotation),
				FailurePolicy: ptr.To(platform.FailurePolicyIgnore),
				Replicas:      ptr.To(int32(3)),
				Enforce:       ptr.To(true),
			},
			ExpectedAnnotations: map[string]string{
				defaults.FeatureDefaultKeyAdmissionControllerEnforce: "true",
			},
		},
		"upgrade: enforce enabled if listenOnUpdates enabled": {
			Spec: platform.SecuredClusterSpec{
				AdmissionControl: &platform.AdmissionControlComponentSpec{
					ListenOnCreates: ptr.To(false),
					ListenOnUpdates: ptr.To(true),
				},
			},
			Status: nonEmptyStatus,
			ExpectedDefaults: &platform.AdmissionControlComponentSpec{
				Bypass:        ptr.To(platform.BypassBreakGlassAnnotation),
				FailurePolicy: ptr.To(platform.FailurePolicyIgnore),
				Replicas:      ptr.To(int32(3)),
				Enforce:       ptr.To(true),
			},
			ExpectedAnnotations: map[string]string{
				defaults.FeatureDefaultKeyAdmissionControllerEnforce: "true",
			},
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
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
				Spec:   *c.Spec.DeepCopy(),
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
			unstructuredSecuredCluster := securedClusterToUnstructured(t, baseSecuredCluster)

			err := reconcileFeatureDefaults(ctx, client, unstructuredSecuredCluster, logr.Discard())
			assert.Nil(t, err, "reconcileFeatureDefaults returned error")

			securedClusterFetched := platform.SecuredCluster{}
			err = client.Get(ctx, ctrlClient.ObjectKey{Namespace: testutils.TestNamespace, Name: clusterName}, &securedClusterFetched)
			assert.Nil(t, err, "retrieving SecuredCluster object from fake Kubernetes client")

			securedClusterDefaults := extractSecuredClusterDefaults(t, unstructuredSecuredCluster)

			// Verify that reconcileFeatureDefaults has modified the admission control defaults as expected.
			assert.Equal(t, securedClusterDefaults.AdmissionControl, c.ExpectedDefaults, "SecuredCluster Defaults do not match expected Defaults")

			// Verify that the expected annotations have been persisted via the provided client.
			securedClusterFetched.Annotations = retainKeys(securedClusterFetched.Annotations, string(defaults.FeatureDefaultKeyAdmissionControllerEnforce))
			assert.Equal(t, securedClusterFetched.Annotations, c.ExpectedAnnotations, "persisted SecuredCluster annotations do not match expected annotations")

			// Verify that the SecuredCluster Spec on the Cluster is unmodified.
			assert.Equal(t, securedClusterFetched.Spec, c.Spec, "persisted SecuredCluster spec is modified")
		})
	}
}

func TestReconcileScannerV4FeatureDefaultsExtension(t *testing.T) {
	cases := map[string]scannerV4DefaultingTestCase{
		"install: auto-sense by default": {
			Spec:   platform.SecuredClusterSpec{},
			Status: platform.SecuredClusterStatus{},
			ExpectedDefaults: &platform.LocalScannerV4ComponentSpec{
				ScannerComponent: &platform.LocalScannerV4AutoSense,
			},
			ExpectedAnnotations: map[string]string{
				defaulting.FeatureDefaultKeyScannerV4: string(platform.LocalScannerV4AutoSense),
			},
		},
		"upgrade: disabled by default": {
			Spec:   platform.SecuredClusterSpec{},
			Status: nonEmptyStatus,
			ExpectedDefaults: &platform.LocalScannerV4ComponentSpec{
				ScannerComponent: &platform.LocalScannerV4Disabled,
			},
			ExpectedAnnotations: map[string]string{
				defaulting.FeatureDefaultKeyScannerV4: string(platform.ScannerV4ComponentDisabled),
			},
		},
		"install: auto-sense explicitly": {
			Spec: platform.SecuredClusterSpec{
				ScannerV4: &platform.LocalScannerV4ComponentSpec{
					ScannerComponent: &platform.LocalScannerV4AutoSense,
				},
			},
			Status:           platform.SecuredClusterStatus{},
			ExpectedDefaults: nil,
		},
		"install: disabled explicitly": {
			Spec: platform.SecuredClusterSpec{
				ScannerV4: &platform.LocalScannerV4ComponentSpec{
					ScannerComponent: &platform.LocalScannerV4Disabled,
				},
			},
			Status:           platform.SecuredClusterStatus{},
			ExpectedDefaults: nil,
		},
		"upgrade: pick up previously persisted default (AutoSense)": {
			Spec:   platform.SecuredClusterSpec{},
			Status: nonEmptyStatus,
			Annotations: map[string]string{
				defaulting.FeatureDefaultKeyScannerV4: string(platform.LocalScannerV4AutoSense),
			},
			ExpectedDefaults: &platform.LocalScannerV4ComponentSpec{
				ScannerComponent: &platform.LocalScannerV4AutoSense,
			},
			ExpectedAnnotations: map[string]string{
				defaulting.FeatureDefaultKeyScannerV4: string(platform.LocalScannerV4AutoSense),
			},
		},
		"upgrade: pick up previously persisted default (Disabled)": {
			Spec:   platform.SecuredClusterSpec{},
			Status: nonEmptyStatus,
			Annotations: map[string]string{
				defaulting.FeatureDefaultKeyScannerV4: string(platform.ScannerV4ComponentDisabled),
			},
			ExpectedDefaults: &platform.LocalScannerV4ComponentSpec{
				ScannerComponent: &platform.LocalScannerV4Disabled,
			},
			ExpectedAnnotations: map[string]string{
				defaulting.FeatureDefaultKeyScannerV4: string(platform.ScannerV4ComponentDisabled),
			},
		},
		"upgrade: ignoring bogus persisted default": {
			Spec:   platform.SecuredClusterSpec{},
			Status: nonEmptyStatus,
			Annotations: map[string]string{
				defaulting.FeatureDefaultKeyScannerV4: "foo",
			},
			ExpectedDefaults: &platform.LocalScannerV4ComponentSpec{
				ScannerComponent: &platform.LocalScannerV4Disabled,
			},
			ExpectedAnnotations: map[string]string{
				defaulting.FeatureDefaultKeyScannerV4: string(platform.ScannerV4ComponentDisabled),
			},
		},
		"previously persisted default is picked up even if status is empty": {
			Spec: platform.SecuredClusterSpec{},
			Annotations: map[string]string{
				defaulting.FeatureDefaultKeyScannerV4: string(platform.ScannerV4ComponentDisabled),
			},
			ExpectedDefaults: &platform.LocalScannerV4ComponentSpec{
				ScannerComponent: &platform.LocalScannerV4Disabled,
			},
			ExpectedAnnotations: map[string]string{
				defaulting.FeatureDefaultKeyScannerV4: string(platform.ScannerV4ComponentDisabled),
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
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
				Spec:   *c.Spec.DeepCopy(),
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
			unstructuredSecuredCluster := securedClusterToUnstructured(t, baseSecuredCluster)

			err := reconcileFeatureDefaults(ctx, client, unstructuredSecuredCluster, logr.Discard())
			assert.Nil(t, err, "reconcileFeatureDefaults returned error")

			securedClusterFetched := platform.SecuredCluster{}
			err = client.Get(ctx, ctrlClient.ObjectKey{Namespace: testutils.TestNamespace, Name: clusterName}, &securedClusterFetched)
			assert.Nil(t, err, "retrieving SecuredCluster object from fake Kubernetes client")

			securedClusterDefaults := extractSecuredClusterDefaults(t, unstructuredSecuredCluster)

			// Verify that reconcileFeatureDefaults has modified the scanner v4 defaults as expected.
			assert.Equal(t, securedClusterDefaults.ScannerV4, c.ExpectedDefaults, "SecuredCluster Defaults do not match expected Defaults")

			// Verify that the expected annotations have been persisted via the provided client.
			securedClusterFetched.Annotations = retainKeys(securedClusterFetched.Annotations, defaulting.FeatureDefaultKeyScannerV4)
			assert.Equal(t, securedClusterFetched.Annotations, c.ExpectedAnnotations, "persisted SecuredCluster annotations do not match expected annotations")

			// Verify that the SecuredCluster Spec on the Cluster is unmodified.
			assert.Equal(t, securedClusterFetched.Spec, c.Spec, "persisted SecuredCluster spec is modified")
		})
	}
}

func retainKeys[T any](m map[string]T, keys ...string) map[string]T {
	for name := range m {
		retain := false
		for _, key := range keys {
			if name == key {
				retain = true
				break
			}
		}
		if !retain {
			delete(m, name)
		}
	}
	if len(m) == 0 {
		m = nil
	}
	return m
}

func securedClusterToUnstructured(t *testing.T, securedCluster *platform.SecuredCluster) *unstructured.Unstructured {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(securedCluster)
	assert.NoError(t, err)
	return &unstructured.Unstructured{Object: obj}
}

func extractSecuredClusterDefaults(t *testing.T, u *unstructured.Unstructured) *platform.SecuredClusterSpec {
	defaults := platform.SecuredClusterSpec{}
	unstructuredSecuredClusterDefaults, ok := u.Object["defaults"].(map[string]interface{})
	if ok {
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredSecuredClusterDefaults, &defaults)
		assert.Nil(t, err, "failed to extract SecuredCluster Defaults from unstructured object")
	}
	return &defaults
}
