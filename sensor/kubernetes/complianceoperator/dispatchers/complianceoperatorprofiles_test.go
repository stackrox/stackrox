package dispatchers

import (
	"testing"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func profileToUnstructured(t *testing.T, p *v1alpha1.Profile) *unstructured.Unstructured {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(p)
	require.NoError(t, err)
	return &unstructured.Unstructured{Object: obj}
}

// TestProfileDispatcher_OpenSCAPProfileID verifies that OpenSCAP profiles use the XCCDF content ID
// (complianceProfile.ID) as ProfileId, so that BuildProfileRefID matches the scan side.
func TestProfileDispatcher_OpenSCAPProfileID(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.ComplianceV2Integrations})
	t.Cleanup(func() { centralcaps.Set(nil) })

	profile := &v1alpha1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ocp4-cis",
			UID:  "some-uid",
			Annotations: map[string]string{
				v1alpha1.ScannerTypeAnnotation: string(v1alpha1.ScannerTypeOpenSCAP),
			},
		},
		ProfilePayload: v1alpha1.ProfilePayload{
			ID: "xccdf_org.ssgproject.content_profile_cis",
		},
	}

	dispatcher := NewProfileDispatcher()
	event := dispatcher.ProcessEvent(profileToUnstructured(t, profile), nil, central.ResourceAction_CREATE_RESOURCE)

	require.NotNil(t, event)
	require.NotEmpty(t, event.ForwardMessages)
	v2Profile := event.ForwardMessages[0].GetComplianceOperatorProfileV2()
	assert.Equal(t, "xccdf_org.ssgproject.content_profile_cis", v2Profile.GetProfileId())
}

// TestProfileDispatcher_CELProfileID verifies that CEL profiles use the k8s object name as
// ProfileId, mirroring what ComplianceScan.Spec.Profile is set to by the compliance operator,
// so that BuildProfileRefID produces matching UUIDs on both the profile and the scan sides.
func TestProfileDispatcher_CELProfileID(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.ComplianceV2Integrations})
	t.Cleanup(func() { centralcaps.Set(nil) })

	profile := &v1alpha1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			// K8s name: what CO puts in ComplianceScan.Spec.Profile for CEL profiles
			Name: "ocp4virt-cis-vm-extension",
			UID:  "cel-uid",
			Annotations: map[string]string{
				v1alpha1.ScannerTypeAnnotation: string(v1alpha1.ScannerTypeCEL),
			},
		},
		ProfilePayload: v1alpha1.ProfilePayload{
			// Short content ID — no XCCDF prefix — which CO does NOT use in spec.profile
			ID: "cis-vm-extension",
		},
	}

	dispatcher := NewProfileDispatcher()
	event := dispatcher.ProcessEvent(profileToUnstructured(t, profile), nil, central.ResourceAction_CREATE_RESOURCE)

	require.NotNil(t, event)
	require.NotEmpty(t, event.ForwardMessages)
	v2Profile := event.ForwardMessages[0].GetComplianceOperatorProfileV2()
	// Must match what the scan dispatcher reads from ComplianceScan.Spec.Profile
	assert.Equal(t, "ocp4virt-cis-vm-extension", v2Profile.GetProfileId())
}

// TestProfileDispatcher_FallbackCases verifies the fallback ProfileId paths added for
// observability: no annotation, an unrecognised scanner type, and a missing XCCDF ID. In every
// case the dispatcher still emits a valid event (never drops it) and logs a Warn.
func TestProfileDispatcher_FallbackCases(t *testing.T) {
	tests := map[string]struct {
		scannerType   string
		xccdfID       string
		name          string
		wantProfileID string
	}{
		"no annotation falls back to XCCDF ID (CO < 1.9)": {
			scannerType:   "",
			xccdfID:       "xccdf_org.ssgproject.content_profile_cis",
			name:          "ocp4-cis",
			wantProfileID: "xccdf_org.ssgproject.content_profile_cis",
		},
		"unrecognised scanner type falls back to XCCDF ID": {
			scannerType:   "WASM", // hypothetical future type
			xccdfID:       "xccdf_org.ssgproject.content_profile_future",
			name:          "future-profile",
			wantProfileID: "xccdf_org.ssgproject.content_profile_future",
		},
		"no XCCDF ID and no annotation yields empty ProfileId": {
			scannerType:   "",
			xccdfID:       "",
			name:          "incomplete-profile",
			wantProfileID: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			centralcaps.Set([]centralsensor.CentralCapability{centralsensor.ComplianceV2Integrations})
			t.Cleanup(func() { centralcaps.Set(nil) })

			profile := &v1alpha1.Profile{
				ObjectMeta: metav1.ObjectMeta{
					Name: tc.name,
					UID:  "some-uid",
					Annotations: map[string]string{
						v1alpha1.ScannerTypeAnnotation: tc.scannerType,
					},
				},
				ProfilePayload: v1alpha1.ProfilePayload{
					ID: tc.xccdfID,
				},
			}

			dispatcher := NewProfileDispatcher()
			event := dispatcher.ProcessEvent(profileToUnstructured(t, profile), nil, central.ResourceAction_CREATE_RESOURCE)

			require.NotNil(t, event)
			require.NotEmpty(t, event.ForwardMessages)
			v2Profile := event.ForwardMessages[0].GetComplianceOperatorProfileV2()
			assert.Equal(t, tc.wantProfileID, v2Profile.GetProfileId())
		})
	}
}

// TestProfileDispatcher_WithoutV2Capability verifies that when ComplianceV2Integrations
// capability is absent, the dispatcher returns nil.
func TestProfileDispatcher_WithoutV2Capability(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{}) // No capabilities
	t.Cleanup(func() { centralcaps.Set(nil) })

	profile := &v1alpha1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ocp4-cis",
			UID:  "some-uid",
		},
		ProfilePayload: v1alpha1.ProfilePayload{
			ID: "xccdf_org.ssgproject.content_profile_cis",
		},
	}

	dispatcher := NewProfileDispatcher()
	event := dispatcher.ProcessEvent(profileToUnstructured(t, profile), nil, central.ResourceAction_CREATE_RESOURCE)

	assert.Nil(t, event, "dispatcher should return nil when ComplianceV2Integrations capability is absent")
}

// TestProfileDispatcher_WithV2Capability verifies that when ComplianceV2Integrations
// capability is present, the dispatcher returns a V2 event.
func TestProfileDispatcher_WithV2Capability(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.ComplianceV2Integrations})
	t.Cleanup(func() { centralcaps.Set(nil) })

	profile := &v1alpha1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ocp4-cis",
			UID:  "some-uid",
		},
		ProfilePayload: v1alpha1.ProfilePayload{
			ID: "xccdf_org.ssgproject.content_profile_cis",
		},
	}

	dispatcher := NewProfileDispatcher()
	event := dispatcher.ProcessEvent(profileToUnstructured(t, profile), nil, central.ResourceAction_CREATE_RESOURCE)

	require.NotNil(t, event)
	require.Len(t, event.ForwardMessages, 1)
	v2Profile := event.ForwardMessages[0].GetComplianceOperatorProfileV2()
	require.NotNil(t, v2Profile)
	assert.Equal(t, "xccdf_org.ssgproject.content_profile_cis", v2Profile.GetProfileId())
}
