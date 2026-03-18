package dispatchers

import (
	"fmt"
	"slices"
	"testing"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/sensor/common/centralcaps"
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

// TestProcessEvent_ExtendsProfile tests rule computation and metadata handling when extending a base profile:
// - effective rules = base rules - disabled rules + enabled rules
// - labels, annotations, description parsed from tailored profile
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

	// Metadata comes from the tailored profile itself, not the base profile.
	assert.Equal(t, "ocp4-tailored", profile.GetLabels()["compliance.openshift.io/profile-bundle"])
	assert.Equal(t, "Platform", profile.GetAnnotations()[v1alpha1.ProductTypeAnnotation])
	assert.Empty(t, profile.GetAnnotations()[v1alpha1.ProductAnnotation])
	assert.Equal(t, "Tailored profile description", profile.GetDescription())

	// Verify rule computation: 3 base - 1 disabled + 1 enabled = 3 rules
	ruleNames := make([]string, len(profile.GetRules()))
	for i, r := range profile.GetRules() {
		ruleNames[i] = r.GetName()
	}
	slices.Sort(ruleNames)
	assert.Equal(t, []string{
		"ocp4-api-server-anonymous-auth",
		"ocp4-api-server-encryption-provider-cipher",
		"ocp4-audit-log-forwarding-enabled",
	}, ruleNames)
}

// TestProcessEvent_StoresTailoredProfileMetadata tests that all metadata fields are stored
// from the tailored profile itself, rather than from the base profile it extends.
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
	require.NotEmpty(t, event.ForwardMessages)
	profile := event.ForwardMessages[0].GetComplianceOperatorProfile()

	assert.Equal(t, "tp-from-scratch", profile.GetName())
	assert.Equal(t, "tp-uid", profile.GetId())
	assert.Equal(t, "xccdf_compliance.openshift.io_profile_tp-from-scratch", profile.GetProfileId())
	assert.Equal(t, "My tailored description", profile.GetDescription())
	assert.Equal(t, "Platform", profile.GetAnnotations()[v1alpha1.ProductTypeAnnotation])
	assert.Equal(t, "some-value", profile.GetLabels()["some-label"])
}

// TestProcessEvent_FromScratch tests that TPs without Extends work: only EnableRules are included,
// with no base profile rules.
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

	dispatcher := NewTailoredProfileDispatcher(newMockProfileLister())
	event := dispatcher.ProcessEvent(toUnstructured(t, tp), nil, central.ResourceAction_CREATE_RESOURCE)

	require.NotNil(t, event)
	require.NotEmpty(t, event.ForwardMessages)
	profile := event.ForwardMessages[0].GetComplianceOperatorProfile()

	ruleNames := make([]string, len(profile.GetRules()))
	for i, r := range profile.GetRules() {
		ruleNames[i] = r.GetName()
	}
	slices.Sort(ruleNames)
	assert.Equal(t, []string{
		"ocp4-api-server-anonymous-auth",
		"ocp4-api-server-encryption-provider-cipher",
	}, ruleNames)
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

// TestProcessEvent_EquivalenceHash verifies that the V2 event carries a non-empty
// equivalence_hash derived from the tailored profile's effective content, and that
// changing any input field (name, namespace, description, title, rules) produces a
// different hash.
func TestProcessEvent_EquivalenceHash(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.ComplianceV2Integrations})
	t.Cleanup(func() { centralcaps.Set(nil) })

	makeTP := func(name, ns, desc, title string, enableRules []string) *v1alpha1.TailoredProfile {
		tp := &v1alpha1.TailoredProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
				UID:       "uid-hash-test",
			},
			Spec: v1alpha1.TailoredProfileSpec{
				Description: desc,
				Title:       title,
			},
			Status: v1alpha1.TailoredProfileStatus{
				ID:    "xccdf_compliance.openshift.io_profile_" + name,
				State: "READY",
			},
		}
		for _, r := range enableRules {
			tp.Spec.EnableRules = append(tp.Spec.EnableRules, v1alpha1.RuleReferenceSpec{Name: r})
		}
		return tp
	}

	getV2Hash := func(t *testing.T, tp *v1alpha1.TailoredProfile) string {
		t.Helper()
		dispatcher := NewTailoredProfileDispatcher(newMockProfileLister())
		event := dispatcher.ProcessEvent(toUnstructured(t, tp), nil, central.ResourceAction_CREATE_RESOURCE)
		require.NotNil(t, event)
		// The V2 event is the second message in ForwardMessages.
		require.Len(t, event.ForwardMessages, 2)
		v2 := event.ForwardMessages[1].GetComplianceOperatorProfileV2()
		require.NotNil(t, v2)
		return v2.GetEquivalenceHash()
	}

	base := makeTP("my-tp", "openshift-compliance", "desc", "title", []string{"rule-a", "rule-b"})
	baseHash := getV2Hash(t, base)
	assert.Len(t, baseHash, 64, "hash should be hex-encoded SHA-256")

	// Expected value: same inputs must produce same hash as the standalone function.
	expected := computeProfileEquivalenceHash("my-tp", "openshift-compliance", "desc", "title", []string{"rule-a", "rule-b"})
	assert.Equal(t, expected, baseHash)

	// Each field change must produce a different hash.
	assert.NotEqual(t, baseHash, getV2Hash(t, makeTP("other-name", "openshift-compliance", "desc", "title", []string{"rule-a", "rule-b"})), "name change")
	assert.NotEqual(t, baseHash, getV2Hash(t, makeTP("my-tp", "other-ns", "desc", "title", []string{"rule-a", "rule-b"})), "namespace change")
	assert.NotEqual(t, baseHash, getV2Hash(t, makeTP("my-tp", "openshift-compliance", "other-desc", "title", []string{"rule-a", "rule-b"})), "description change")
	assert.NotEqual(t, baseHash, getV2Hash(t, makeTP("my-tp", "openshift-compliance", "desc", "other-title", []string{"rule-a", "rule-b"})), "title change")
	assert.NotEqual(t, baseHash, getV2Hash(t, makeTP("my-tp", "openshift-compliance", "desc", "title", []string{"rule-a", "rule-c"})), "rule change")

	// Rule order must not affect the hash.
	h1 := getV2Hash(t, makeTP("my-tp", "openshift-compliance", "desc", "title", []string{"rule-a", "rule-b"}))
	h2 := getV2Hash(t, makeTP("my-tp", "openshift-compliance", "desc", "title", []string{"rule-b", "rule-a"}))
	assert.Equal(t, h1, h2, "rule order should not affect hash")
}

// TestProcessEvent_NonTailoredProfileNoHash verifies that non-tailored profiles do not set an equivalence hash.
func TestProcessEvent_NonTailoredProfileNoHash(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.ComplianceV2Integrations})
	t.Cleanup(func() { centralcaps.Set(nil) })

	// Use the non-tailored profile dispatcher (ProfileDispatcher).
	profile := &v1alpha1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ocp4-cis",
			Namespace: "openshift-compliance",
			UID:       "standard-uid",
		},
		ProfilePayload: v1alpha1.ProfilePayload{
			ID:    "xccdf_org.ssgproject.content_profile_cis",
			Title: "CIS Benchmark",
		},
	}
	dispatcher := NewProfileDispatcher()
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(profile)
	require.NoError(t, err)
	event := dispatcher.ProcessEvent(&unstructured.Unstructured{Object: unstructuredObj}, nil, central.ResourceAction_CREATE_RESOURCE)
	require.NotNil(t, event)
	require.Len(t, event.ForwardMessages, 2)
	v2 := event.ForwardMessages[1].GetComplianceOperatorProfileV2()
	require.NotNil(t, v2)
	assert.Empty(t, v2.GetEquivalenceHash(), "non-tailored profiles must not carry an equivalence_hash")
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

// TestProcessEvent_NoV2CentralCapability tests that when ComplianceV2TailoredProfiles is absent we only send the
// Compliance V1 event
func TestProcessEvent_NoV2CentralCapability(t *testing.T) {
	// ComplianceV2Integrations present, ComplianceV2TailoredProfiles absent.
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.ComplianceV2Integrations})
	t.Cleanup(func() { centralcaps.Set(nil) })

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
	require.Len(t, event.ForwardMessages, 1) // V1 only, no V2
	assert.NotNil(t, event.ForwardMessages[0].GetComplianceOperatorProfile())
}

// TestProcessEvent_V2EventHasTailoredProfileKind tests that when both ComplianceV2TailoredProfiles
// and ComplianceV2Integrations capabilities are present we send both V1 and V2 events and the V2 event carries
// OperatorKind TAILORED_PROFILE.
func TestProcessEvent_V2EventHasTailoredProfileKind(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.ComplianceV2Integrations, centralsensor.ComplianceV2TailoredProfiles})
	t.Cleanup(func() { centralcaps.Set(nil) })

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
	require.Len(t, event.ForwardMessages, 2) // V1 + V2

	v2Profile := event.ForwardMessages[1].GetComplianceOperatorProfileV2()
	require.NotNil(t, v2Profile)
	assert.Equal(t, central.ComplianceOperatorProfileV2_TAILORED_PROFILE, v2Profile.GetOperatorKind())
}
