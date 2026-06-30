package dispatchers

import (
	"fmt"
	"slices"
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
			Annotations: map[string]string{
				v1alpha1.ProductTypeAnnotation: "Platform",
			},
			Labels: map[string]string{
				"compliance.openshift.io/profile-bundle": "ocp4-tailored",
			},
		},
		Spec: v1alpha1.TailoredProfileSpec{
			Description: "Tailored profile description",
			Extends:     "ocp4-cis",
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
	require.Len(t, event.ForwardMessages, 1)

	profile := event.ForwardMessages[0].GetComplianceOperatorProfileV2()
	require.NotNil(t, profile)

	assert.Equal(t, "ocp4-tailored", profile.GetLabels()["compliance.openshift.io/profile-bundle"])
	assert.Equal(t, "Platform", profile.GetAnnotations()[v1alpha1.ProductTypeAnnotation])
	assert.Empty(t, profile.GetAnnotations()[v1alpha1.ProductAnnotation])
	assert.Equal(t, "Tailored profile description", profile.GetDescription())
	assert.Equal(t, central.ComplianceOperatorProfileV2_TAILORED_PROFILE, profile.GetOperatorKind())

	ruleNames := make([]string, len(profile.GetRules()))
	for i, r := range profile.GetRules() {
		ruleNames[i] = r.GetRuleName()
	}
	slices.Sort(ruleNames)
	assert.Equal(t, []string{
		"ocp4-api-server-anonymous-auth",
		"ocp4-api-server-encryption-provider-cipher",
		"ocp4-audit-log-forwarding-enabled",
	}, ruleNames)
}

func TestProcessEvent_StoresTailoredProfileMetadata(t *testing.T) {
	tp := &v1alpha1.TailoredProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tp-from-scratch",
			Namespace: "openshift-compliance",
			UID:       "tp-uid",
			Annotations: map[string]string{
				v1alpha1.ProductTypeAnnotation: "Platform",
			},
			Labels: map[string]string{
				"some-label": "some-value",
			},
		},
		Spec: v1alpha1.TailoredProfileSpec{
			Description: "My tailored description",
			EnableRules: []v1alpha1.RuleReferenceSpec{
				{Name: "ocp4-api-server-anonymous-auth"},
			},
		},
		Status: v1alpha1.TailoredProfileStatus{
			ID:    "xccdf_compliance.openshift.io_profile_tp-from-scratch",
			State: "READY",
		},
	}

	dispatcher := NewTailoredProfileDispatcher(newMockProfileLister())
	event := dispatcher.ProcessEvent(toUnstructured(t, tp), nil, central.ResourceAction_CREATE_RESOURCE)

	require.NotNil(t, event)
	require.Len(t, event.ForwardMessages, 1)

	profile := event.ForwardMessages[0].GetComplianceOperatorProfileV2()
	require.NotNil(t, profile)
	assert.Equal(t, "tp-from-scratch", profile.GetName())
	assert.Equal(t, "tp-uid", profile.GetId())
	assert.Equal(t, "xccdf_compliance.openshift.io_profile_tp-from-scratch", profile.GetProfileId())
	assert.Equal(t, "My tailored description", profile.GetDescription())
	assert.Equal(t, "Platform", profile.GetAnnotations()[v1alpha1.ProductTypeAnnotation])
	assert.Equal(t, "some-value", profile.GetLabels()["some-label"])
}

func TestProcessEvent_FromScratch(t *testing.T) {
	tp := &v1alpha1.TailoredProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tp-from-scratch",
			Namespace: "openshift-compliance",
			UID:       "tp-uid",
		},
		Spec: v1alpha1.TailoredProfileSpec{
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

	dispatcher := NewTailoredProfileDispatcher(newMockProfileLister())
	event := dispatcher.ProcessEvent(toUnstructured(t, tp), nil, central.ResourceAction_CREATE_RESOURCE)

	require.NotNil(t, event)
	require.Len(t, event.ForwardMessages, 1)

	profile := event.ForwardMessages[0].GetComplianceOperatorProfileV2()
	require.NotNil(t, profile)

	ruleNames := make([]string, len(profile.GetRules()))
	for i, r := range profile.GetRules() {
		ruleNames[i] = r.GetRuleName()
	}
	slices.Sort(ruleNames)
	assert.Equal(t, []string{
		"ocp4-api-server-anonymous-auth",
		"ocp4-api-server-encryption-provider-cipher",
	}, ruleNames)
}

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
			State: "PENDING",
		},
	}

	dispatcher := NewTailoredProfileDispatcher(newMockProfileLister())
	event := dispatcher.ProcessEvent(toUnstructured(t, tp), nil, central.ResourceAction_CREATE_RESOURCE)

	assert.Nil(t, event)
}

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

func TestProcessEvent_V2EventHasTailoredProfileKind(t *testing.T) {
	tp := &v1alpha1.TailoredProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ocp4-cis-tailored",
			Namespace: "openshift-compliance",
			UID:       "tp-uid",
		},
		Spec: v1alpha1.TailoredProfileSpec{
			EnableRules: []v1alpha1.RuleReferenceSpec{{Name: "some-rule"}},
		},
		Status: v1alpha1.TailoredProfileStatus{
			ID:    "xccdf_compliance.openshift.io_profile_ocp4-cis-tailored",
			State: "READY",
		},
	}

	dispatcher := NewTailoredProfileDispatcher(newMockProfileLister())
	event := dispatcher.ProcessEvent(toUnstructured(t, tp), nil, central.ResourceAction_CREATE_RESOURCE)

	require.NotNil(t, event)
	require.Len(t, event.ForwardMessages, 1)

	v2Profile := event.ForwardMessages[0].GetComplianceOperatorProfileV2()
	require.NotNil(t, v2Profile)
	assert.Equal(t, central.ComplianceOperatorProfileV2_TAILORED_PROFILE, v2Profile.GetOperatorKind())
}
