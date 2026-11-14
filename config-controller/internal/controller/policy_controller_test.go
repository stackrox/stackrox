package controller

import (
	"context"
	"testing"
	"time"

	configstackroxiov1alpha1 "github.com/stackrox/rox/config-controller/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var baseTime = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

func makeCR(name, policyName, uid string, created time.Time, deleting bool) *configstackroxiov1alpha1.SecurityPolicy {
	cr := &configstackroxiov1alpha1.SecurityPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         "stackrox",
			UID:               types.UID(uid),
			CreationTimestamp: metav1.NewTime(created),
		},
		Spec: configstackroxiov1alpha1.SecurityPolicySpec{PolicyName: policyName},
	}
	if deleting {
		ts := metav1.NewTime(created)
		cr.ObjectMeta.DeletionTimestamp = &ts
		// A fake client requires a finalizer on objects created with a deletion timestamp.
		cr.ObjectMeta.Finalizers = []string{policyFinalizer}
	}
	return cr
}

func newTestReconciler(t *testing.T, objs ...runtime.Object) *SecurityPolicyReconciler {
	scheme := runtime.NewScheme()
	require.NoError(t, configstackroxiov1alpha1.AddToScheme(scheme))
	cl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objs...).Build()
	return &SecurityPolicyReconciler{K8sClient: cl}
}

// TestFindConflictingSeniorCR validates policyName uniqueness resolution across CRs.
func TestFindConflictingSeniorCR(t *testing.T) {
	cases := map[string]struct {
		subject      *configstackroxiov1alpha1.SecurityPolicy
		others       []runtime.Object
		expectConfli bool
		expectName   string
	}{
		"no other CRs - may keep name": {
			subject:      makeCR("cr1", "Test-Policy-A", "uid-1", baseTime, false),
			others:       nil,
			expectConfli: false,
		},
		"different policyName - no conflict": {
			subject: makeCR("cr2", "Test-Policy-B", "uid-2", baseTime.Add(time.Hour), false),
			others: []runtime.Object{
				makeCR("cr1", "Test-Policy-A", "uid-1", baseTime, false),
			},
			expectConfli: false,
		},
		"junior yields to senior": {
			subject: makeCR("cr2", "Test-Policy-A", "uid-2", baseTime.Add(time.Hour), false),
			others: []runtime.Object{
				makeCR("cr1", "Test-Policy-A", "uid-1", baseTime, false),
			},
			expectConfli: true,
			expectName:   "cr1",
		},
		"senior keeps name over junior": {
			subject: makeCR("cr1", "Test-Policy-A", "uid-1", baseTime, false),
			others: []runtime.Object{
				makeCR("cr2", "Test-Policy-A", "uid-2", baseTime.Add(time.Hour), false),
			},
			expectConfli: false,
		},
		"same timestamp broken by smaller UID": {
			subject: makeCR("cr-b", "Test-Policy-A", "uid-b", baseTime, false),
			others: []runtime.Object{
				makeCR("cr-a", "Test-Policy-A", "uid-a", baseTime, false),
			},
			expectConfli: true,
			expectName:   "cr-a",
		},
		"senior CR being deleted still wins until fully removed": {
			// A senior CR mid-deletion may still own its Central policy (finalizer
			// pending), so the junior must keep yielding rather than adopt it.
			subject: makeCR("cr2", "Test-Policy-A", "uid-2", baseTime.Add(time.Hour), false),
			others: []runtime.Object{
				makeCR("cr1", "Test-Policy-A", "uid-1", baseTime, true),
			},
			expectConfli: true,
			expectName:   "cr1",
		},
		"junior CR being deleted does not make senior yield": {
			subject: makeCR("cr1", "Test-Policy-A", "uid-1", baseTime, false),
			others: []runtime.Object{
				makeCR("cr2", "Test-Policy-A", "uid-2", baseTime.Add(time.Hour), true),
			},
			expectConfli: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			objs := append([]runtime.Object{tc.subject}, tc.others...)
			r := newTestReconciler(t, objs...)

			conflicting, err := r.findConflictingSeniorCR(context.Background(), tc.subject)
			require.NoError(t, err)

			if tc.expectConfli {
				require.NotNil(t, conflicting, "expected a conflicting senior CR")
				assert.Equal(t, tc.expectName, conflicting.GetName())
			} else {
				assert.Nil(t, conflicting, "expected no conflict")
			}
		})
	}
}

func withPolicyID(cr *configstackroxiov1alpha1.SecurityPolicy, policyID string) *configstackroxiov1alpha1.SecurityPolicy {
	cr.Status.PolicyId = policyID
	return cr
}

// TestAnotherCRSharesPolicyID validates the deletion-path ownership guard that
// prevents a CR from deleting a Central policy another live CR still references.
func TestAnotherCRSharesPolicyID(t *testing.T) {
	cases := map[string]struct {
		subject      *configstackroxiov1alpha1.SecurityPolicy
		others       []runtime.Object
		expectShared bool
	}{
		"no recorded policy id": {
			subject:      withPolicyID(makeCR("cr1", "Test-Policy-A", "uid-1", baseTime, false), ""),
			others:       nil,
			expectShared: false,
		},
		"no other CRs": {
			subject:      withPolicyID(makeCR("cr1", "Test-Policy-A", "uid-1", baseTime, false), "pol-1"),
			others:       nil,
			expectShared: false,
		},
		"distinct policy ids": {
			subject: withPolicyID(makeCR("cr1", "Test-Policy-A", "uid-1", baseTime, false), "pol-1"),
			others: []runtime.Object{
				withPolicyID(makeCR("cr2", "Test-Policy-B", "uid-2", baseTime, false), "pol-2"),
			},
			expectShared: false,
		},
		"shared policy id with live CR": {
			subject: withPolicyID(makeCR("cr1", "Test-Policy-A", "uid-1", baseTime, false), "pol-1"),
			others: []runtime.Object{
				withPolicyID(makeCR("cr2", "Test-Policy-A", "uid-2", baseTime, false), "pol-1"),
			},
			expectShared: true,
		},
		"shared policy id but other CR is being deleted": {
			subject: withPolicyID(makeCR("cr1", "Test-Policy-A", "uid-1", baseTime, false), "pol-1"),
			others: []runtime.Object{
				withPolicyID(makeCR("cr2", "Test-Policy-A", "uid-2", baseTime, true), "pol-1"),
			},
			expectShared: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			objs := append([]runtime.Object{tc.subject}, tc.others...)
			r := newTestReconciler(t, objs...)

			shared, err := r.anotherCRSharesPolicyID(context.Background(), tc.subject)
			require.NoError(t, err)
			assert.Equal(t, tc.expectShared, shared)
		})
	}
}
