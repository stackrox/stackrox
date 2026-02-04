package dispatchers

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

// mockGenericLister implements cache.GenericLister for testing
type mockGenericLister struct {
	objects map[string]map[string]runtime.Object // namespace -> name -> object
}

func newMockGenericLister() *mockGenericLister {
	return &mockGenericLister{
		objects: make(map[string]map[string]runtime.Object),
	}
}

func (m *mockGenericLister) addTailoredProfile(tp *v1alpha1.TailoredProfile) error {
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(tp)
	if err != nil {
		return err
	}
	u := &unstructured.Unstructured{Object: unstructuredObj}

	ns := tp.Namespace
	if m.objects[ns] == nil {
		m.objects[ns] = make(map[string]runtime.Object)
	}
	m.objects[ns][tp.Name] = u
	return nil
}

func (m *mockGenericLister) List(selector labels.Selector) ([]runtime.Object, error) {
	return nil, nil
}

func (m *mockGenericLister) Get(name string) (runtime.Object, error) {
	return nil, errors.New("not found")
}

func (m *mockGenericLister) ByNamespace(namespace string) cache.GenericNamespaceLister {
	return &mockGenericNamespaceLister{
		namespace: namespace,
		objects:   m.objects[namespace],
	}
}

type mockGenericNamespaceLister struct {
	namespace string
	objects   map[string]runtime.Object
}

func (m *mockGenericNamespaceLister) List(selector labels.Selector) ([]runtime.Object, error) {
	return nil, nil
}

func (m *mockGenericNamespaceLister) Get(name string) (runtime.Object, error) {
	if m.objects == nil {
		return nil, fmt.Errorf("not found: %s", name)
	}
	obj, found := m.objects[name]
	if !found {
		return nil, fmt.Errorf("not found: %s", name)
	}
	return obj, nil
}

func TestGetProfileIDForScan(t *testing.T) {
	testCases := []struct {
		name             string
		scan             *v1alpha1.ComplianceScan
		tailoredProfiles []*v1alpha1.TailoredProfile
		expectedID       string
	}{
		{
			name: "OpenSCAP scan with XCCDF ID returns as-is",
			scan: &v1alpha1.ComplianceScan{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ocp4-cis",
					Namespace: "openshift-compliance",
				},
				Spec: v1alpha1.ComplianceScanSpec{
					Profile: "xccdf_org.ssgproject.content_profile_cis",
				},
			},
			tailoredProfiles: nil,
			expectedID:       "xccdf_org.ssgproject.content_profile_cis",
		},
		{
			name: "TailoredProfile scan with XCCDF ID returns as-is",
			scan: &v1alpha1.ComplianceScan{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tp-scan",
					Namespace: "openshift-compliance",
				},
				Spec: v1alpha1.ComplianceScanSpec{
					Profile: "xccdf_compliance.openshift.io_profile_my-tailored-profile",
				},
			},
			tailoredProfiles: nil,
			expectedID:       "xccdf_compliance.openshift.io_profile_my-tailored-profile",
		},
		{
			name: "CEL scan with profile NAME looks up TailoredProfile and returns XCCDF ID",
			scan: &v1alpha1.ComplianceScan{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "custom-checks-only",
					Namespace: "openshift-compliance",
				},
				Spec: v1alpha1.ComplianceScanSpec{
					Profile: "custom-checks-only",
				},
			},
			tailoredProfiles: []*v1alpha1.TailoredProfile{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "custom-checks-only",
						Namespace: "openshift-compliance",
					},
					Status: v1alpha1.TailoredProfileStatus{
						ID:    "xccdf_compliance.openshift.io_profile_custom-checks-only",
						State: v1alpha1.TailoredProfileStateReady,
					},
				},
			},
			expectedID: "xccdf_compliance.openshift.io_profile_custom-checks-only",
		},
		{
			name: "CEL scan with profile NAME but TailoredProfile not found returns original",
			scan: &v1alpha1.ComplianceScan{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unknown-scan",
					Namespace: "openshift-compliance",
				},
				Spec: v1alpha1.ComplianceScanSpec{
					Profile: "unknown-profile",
				},
			},
			tailoredProfiles: nil,
			expectedID:       "unknown-profile",
		},
		{
			name: "CEL scan with profile NAME but TailoredProfile has empty status.id returns original",
			scan: &v1alpha1.ComplianceScan{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pending-scan",
					Namespace: "openshift-compliance",
				},
				Spec: v1alpha1.ComplianceScanSpec{
					Profile: "pending-profile",
				},
			},
			tailoredProfiles: []*v1alpha1.TailoredProfile{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pending-profile",
						Namespace: "openshift-compliance",
					},
					Status: v1alpha1.TailoredProfileStatus{
						ID:    "", // Not yet assigned
						State: v1alpha1.TailoredProfileStatePending,
					},
				},
			},
			expectedID: "pending-profile",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock lister
			lister := newMockGenericLister()
			for _, tp := range tc.tailoredProfiles {
				err := lister.addTailoredProfile(tp)
				require.NoError(t, err)
			}

			// Create dispatcher with mock lister
			dispatcher := NewScanDispatcher(lister)

			// Call the method under test
			result := dispatcher.getProfileIDForScan(tc.scan)

			// Assert
			assert.Equal(t, tc.expectedID, result)
		})
	}
}

func TestGetProfileIDForScan_NilLister(t *testing.T) {
	// When lister is nil, should return original profile ID
	dispatcher := NewScanDispatcher(nil)

	scan := &v1alpha1.ComplianceScan{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cel-scan",
			Namespace: "openshift-compliance",
		},
		Spec: v1alpha1.ComplianceScanSpec{
			Profile: "custom-checks-only", // Would normally trigger lookup
		},
	}

	result := dispatcher.getProfileIDForScan(scan)
	assert.Equal(t, "custom-checks-only", result)
}
