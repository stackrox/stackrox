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

// TestProfileDispatcher_ProfileIDSelection verifies ProfileId selection for every scanner-type
// annotation value, so that BuildProfileRefID produces matching UUIDs on both the profile and
// the scan sides. Unrecognised scanner types and empty XCCDF IDs still produce a valid event
// (not dropped) — the dispatcher only logs a Warn in those cases.
func TestProfileDispatcher_ProfileIDSelection(t *testing.T) {
	tests := map[string]struct {
		scannerType   string
		xccdfID       string
		name          string
		wantProfileID string
	}{
		"OpenSCAP uses XCCDF content ID": {
			scannerType:   string(v1alpha1.ScannerTypeOpenSCAP),
			xccdfID:       "xccdf_org.ssgproject.content_profile_cis",
			name:          "ocp4-cis",
			wantProfileID: "xccdf_org.ssgproject.content_profile_cis",
		},
		"CEL uses k8s object name": {
			scannerType: string(v1alpha1.ScannerTypeCEL),
			// Short content ID — no XCCDF prefix — which CO does NOT use in spec.profile
			xccdfID:       "cis-vm-extension",
			name:          "ocp4virt-cis-vm-extension",
			wantProfileID: "ocp4virt-cis-vm-extension",
		},
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
			v1Profile := event.ForwardMessages[0].GetComplianceOperatorProfile()
			assert.Equal(t, tc.wantProfileID, v1Profile.GetProfileId())
		})
	}
}

// TestProfileDispatcher_CELProfileID_V2 verifies the V2 event also carries the k8s name as ProfileId.
func TestProfileDispatcher_CELProfileID_V2(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.ComplianceV2Integrations})
	t.Cleanup(func() { centralcaps.Set(nil) })

	profile := &v1alpha1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ocp4virt-cis-vm-extension",
			UID:  "cel-uid",
			Annotations: map[string]string{
				v1alpha1.ScannerTypeAnnotation: string(v1alpha1.ScannerTypeCEL),
			},
		},
		ProfilePayload: v1alpha1.ProfilePayload{
			ID: "cis-vm-extension",
		},
	}

	dispatcher := NewProfileDispatcher()
	event := dispatcher.ProcessEvent(profileToUnstructured(t, profile), nil, central.ResourceAction_CREATE_RESOURCE)

	require.NotNil(t, event)
	require.Len(t, event.ForwardMessages, 2)

	v2Profile := event.ForwardMessages[1].GetComplianceOperatorProfileV2()
	require.NotNil(t, v2Profile)
	assert.Equal(t, "ocp4virt-cis-vm-extension", v2Profile.GetProfileId())
	assert.Equal(t, central.ComplianceOperatorProfileV2_PROFILE, v2Profile.GetOperatorKind())
}
