package dispatchers

import (
	"fmt"
	"testing"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

// mockNamespaceLister implements cache.GenericNamespaceLister
type mockNamespaceLister struct {
	objects map[string]runtime.Object
}

func (m *mockNamespaceLister) List(_ labels.Selector) ([]runtime.Object, error) {
	panic("List should not be called in these tests")
}

func (m *mockNamespaceLister) Get(name string) (runtime.Object, error) {
	if obj, ok := m.objects[name]; ok {
		return obj, nil
	}
	return nil, fmt.Errorf("not found: %s", name)
}

// mockProfileLister implements cache.GenericLister
type mockProfileLister struct {
	namespaces map[string]*mockNamespaceLister
}

func (m *mockProfileLister) List(_ labels.Selector) ([]runtime.Object, error) {
	panic("List should not be called in these tests")
}

func (m *mockProfileLister) Get(_ string) (runtime.Object, error) {
	panic("Get should not be called in these tests")
}

func (m *mockProfileLister) ByNamespace(ns string) cache.GenericNamespaceLister {
	if nsl, ok := m.namespaces[ns]; ok {
		return nsl
	}
	return &mockNamespaceLister{objects: map[string]runtime.Object{}}
}

func newMockProfileLister() *mockProfileLister {
	return &mockProfileLister{
		namespaces: make(map[string]*mockNamespaceLister),
	}
}

func (m *mockProfileLister) addProfile(namespace string, profile *v1alpha1.Profile) {
	if m.namespaces[namespace] == nil {
		m.namespaces[namespace] = &mockNamespaceLister{
			objects: make(map[string]runtime.Object),
		}
	}
	unstructuredObj, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(profile)
	m.namespaces[namespace].objects[profile.Name] = &unstructured.Unstructured{Object: unstructuredObj}
}

func toUnstructured(t *testing.T, tp *v1alpha1.TailoredProfile) *unstructured.Unstructured {
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(tp)
	require.NoError(t, err)
	return &unstructured.Unstructured{Object: unstructuredObj}
}

// TestProcessEvent_ExtendsProfile tests rule computation and metadata inheritance when extending a base profile:
// - effective rules = base rules - disabled rules + enabled rules
// - labels, annotations, description inherited from base profile
func TestProcessEvent_ExtendsProfile(t *testing.T) {
	baseProfile := &v1alpha1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ocp4-cis",
			Namespace: "openshift-compliance",
			Labels: map[string]string{
				"compliance.openshift.io/profile-bundle": "ocp4",
			},
			Annotations: map[string]string{
				v1alpha1.ProductTypeAnnotation: "Platform",
				v1alpha1.ProductAnnotation:     "ocp4",
			},
		},
		ProfilePayload: v1alpha1.ProfilePayload{
			ID:          "xccdf_org.ssgproject.content_profile_cis",
			Description: "Base profile description from CIS benchmark",
			Rules: []v1alpha1.ProfileRule{
				"ocp4-api-server-anonymous-auth",
				"ocp4-api-server-audit-log-path",
				"ocp4-api-server-encryption-provider-cipher",
			},
		},
	}

	lister := newMockProfileLister()
	lister.addProfile("openshift-compliance", baseProfile)

	tp := &v1alpha1.TailoredProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ocp4-cis-tailored",
			Namespace: "openshift-compliance",
			UID:       "tp-uid",
		},
		Spec: v1alpha1.TailoredProfileSpec{
			Extends: "ocp4-cis",
			// Description intentionally empty to test inheritance
			DisableRules: []v1alpha1.RuleReferenceSpec{
				{Name: "ocp4-api-server-audit-log-path"},
			},
			EnableRules: []v1alpha1.RuleReferenceSpec{
				{Name: "ocp4-audit-log-forwarding-enabled"},
			},
		},
		Status: v1alpha1.TailoredProfileStatus{
			ID:    "xccdf_compliance.openshift.io_profile_ocp4-cis-tailored",
			State: "READY",
		},
	}

	dispatcher := NewTailoredProfileDispatcher(lister)
	event := dispatcher.ProcessEvent(toUnstructured(t, tp), nil, central.ResourceAction_CREATE_RESOURCE)

	require.NotNil(t, event)
	require.NotEmpty(t, event.ForwardMessages)
	profile := event.ForwardMessages[0].GetComplianceOperatorProfile()

	// Verify metadata inheritance from base profile
	assert.Equal(t, "ocp4", profile.GetLabels()["compliance.openshift.io/profile-bundle"])
	assert.Equal(t, "Platform", profile.GetAnnotations()[v1alpha1.ProductTypeAnnotation])
	assert.Equal(t, "ocp4", profile.GetAnnotations()[v1alpha1.ProductAnnotation])
	assert.Equal(t, "Base profile description from CIS benchmark", profile.GetDescription())

	// Verify rule computation: 3 base - 1 disabled + 1 enabled = 3 rules
	ruleNames := make([]string, len(profile.GetRules()))
	for i, r := range profile.GetRules() {
		ruleNames[i] = r.GetName()
	}
	assert.Len(t, ruleNames, 3)
	assert.Contains(t, ruleNames, "ocp4-api-server-anonymous-auth")
	assert.Contains(t, ruleNames, "ocp4-api-server-encryption-provider-cipher")
	assert.Contains(t, ruleNames, "ocp4-audit-log-forwarding-enabled")
	assert.NotContains(t, ruleNames, "ocp4-api-server-audit-log-path")
}

// TestProcessEvent_FromScratch tests that TPs without Extends work (only EnableRules)
func TestProcessEvent_FromScratch(t *testing.T) {
	tp := &v1alpha1.TailoredProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tp-from-scratch",
			Namespace: "openshift-compliance",
			UID:       "tp-uid",
		},
		Spec: v1alpha1.TailoredProfileSpec{
			// No Extends
			EnableRules: []v1alpha1.RuleReferenceSpec{
				{Name: "ocp4-api-server-anonymous-auth"},
				{Name: "ocp4-api-server-encryption-provider-cipher"},
			},
		},
		Status: v1alpha1.TailoredProfileStatus{
			ID:    "xccdf_compliance.openshift.io_profile_tp-from-scratch",
			State: "READY",
		},
	}

	lister := newMockProfileLister() // No base profile needed
	dispatcher := NewTailoredProfileDispatcher(lister)
	event := dispatcher.ProcessEvent(toUnstructured(t, tp), nil, central.ResourceAction_CREATE_RESOURCE)

	require.NotNil(t, event)
	require.NotEmpty(t, event.ForwardMessages)
	profile := event.ForwardMessages[0].GetComplianceOperatorProfile()

	// Only enabled rules should be present
	ruleNames := make([]string, len(profile.GetRules()))
	for i, r := range profile.GetRules() {
		ruleNames[i] = r.GetName()
	}
	assert.Len(t, profile.GetRules(), 2)
	assert.Contains(t, ruleNames, "ocp4-api-server-anonymous-auth")
	assert.Contains(t, ruleNames, "ocp4-api-server-encryption-provider-cipher")
}

// TestProcessEvent_NoStatusID tests that non-ready TPs (no Status.ID) are skipped
func TestProcessEvent_NoStatusID(t *testing.T) {
	tp := &v1alpha1.TailoredProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pending-profile",
			Namespace: "openshift-compliance",
			UID:       "tp-uid",
		},
		Spec: v1alpha1.TailoredProfileSpec{
			Extends: "ocp4-cis",
		},
		Status: v1alpha1.TailoredProfileStatus{
			// ID is empty - not yet processed by CO
			State: "PENDING",
		},
	}

	dispatcher := NewTailoredProfileDispatcher(newMockProfileLister())
	event := dispatcher.ProcessEvent(toUnstructured(t, tp), nil, central.ResourceAction_CREATE_RESOURCE)

	assert.Nil(t, event)
}

// TestProcessEvent_BaseProfileNotFound tests that TPs with missing base profile are skipped
func TestProcessEvent_BaseProfileNotFound(t *testing.T) {
	tp := &v1alpha1.TailoredProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "orphan-profile",
			Namespace: "openshift-compliance",
			UID:       "tp-uid",
		},
		Spec: v1alpha1.TailoredProfileSpec{
			Extends: "non-existent-profile",
		},
		Status: v1alpha1.TailoredProfileStatus{
			ID:    "xccdf_compliance.openshift.io_profile_orphan-profile",
			State: "READY",
		},
	}

	dispatcher := NewTailoredProfileDispatcher(newMockProfileLister())
	event := dispatcher.ProcessEvent(toUnstructured(t, tp), nil, central.ResourceAction_CREATE_RESOURCE)

	assert.Nil(t, event)
}
