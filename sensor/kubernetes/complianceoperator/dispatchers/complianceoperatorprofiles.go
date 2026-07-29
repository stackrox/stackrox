package dispatchers

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	log = logging.LoggerForModule()
)

// ProfileDispatcher handles compliance operator profile objects
type ProfileDispatcher struct {
}

// NewProfileDispatcher creates and returns a new profile dispatcher
func NewProfileDispatcher() *ProfileDispatcher {
	return &ProfileDispatcher{}
}

// ProcessEvent processes a compliance operator profile
func (c *ProfileDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	var complianceProfile v1alpha1.Profile

	unstructuredObject, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Not of type 'unstructured': %T", obj)
		return nil
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.Object, &complianceProfile); err != nil {
		log.Errorf("error converting unstructured to compliance profile: %v", err)
		return nil
	}

	// For nextgen compliance this ID is used to tell us which clusters have which profiles.  It is also
	// useful for the deduping from sensor.
	uid := string(complianceProfile.UID)

	// Profiles using the CEL scanner (annotation compliance.openshift.io/scanner-type=CEL) have no
	// XCCDF profile ID, so the compliance operator sets ComplianceScan.Spec.Profile to the profile's
	// k8s object name instead. We must use the same value as ProfileId here so that BuildProfileRefID
	// produces matching UUIDs on both the profile and the scan sides. The same pattern was applied to
	// CEL-based tailored profiles in https://github.com/stackrox/stackrox/pull/21905 — see
	// complianceoperatortailoredprofiles.go.
	profileID := complianceProfile.ID
	switch scannerType := complianceProfile.GetAnnotations()[v1alpha1.ScannerTypeAnnotation]; scannerType {
	case string(v1alpha1.ScannerTypeCEL):
		profileID = complianceProfile.Name
		log.Debugf("Profile %s uses CEL scanner: using k8s name %q as ProfileId instead of XCCDF ID %q",
			complianceProfile.Name, profileID, complianceProfile.ID)
	case string(v1alpha1.ScannerTypeOpenSCAP), "":
		// OpenSCAP and unannotated profiles use the XCCDF content ID — no action needed.
	default:
		// Unknown scanner type: CO may have introduced a new type that also uses k8s name in
		// ComplianceScan.Spec.Profile. If compliance coverage shows 0 results for this profile,
		// check whether the scan's spec.profile matches the XCCDF ID or the k8s name and update
		// this dispatcher accordingly.
		log.Warnf("Profile %s has unrecognised scanner-type annotation %q: using XCCDF ID %q as ProfileId; "+
			"if compliance coverage shows 0 results, this scanner type may need handling here",
			complianceProfile.Name, scannerType, profileID)
	}
	if profileID == "" {
		log.Warnf("Profile %s has an empty ProfileId; compliance coverage results will be missing for this profile",
			complianceProfile.Name)
	}

	protoProfile := &storage.ComplianceOperatorProfile{
		Id:          uid,
		ProfileId:   profileID,
		Name:        complianceProfile.Name,
		Labels:      complianceProfile.Labels,
		Annotations: complianceProfile.Annotations,
		Description: complianceProfile.Description,
	}
	for _, r := range complianceProfile.Rules {
		protoProfile.Rules = append(protoProfile.Rules, &storage.ComplianceOperatorProfile_Rule{
			Name: string(r),
		})
	}

	events := []*central.SensorEvent{
		{
			Id:     uid,
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorProfile{
				ComplianceOperatorProfile: protoProfile,
			},
		},
	}

	if centralcaps.Has(centralsensor.ComplianceV2Integrations) {
		protoProfile := &central.ComplianceOperatorProfileV2{
			Id:             uid,
			ProfileId:      profileID,
			Name:           complianceProfile.Name,
			ProfileVersion: complianceProfile.Version,
			Labels:         complianceProfile.Labels,
			Annotations:    complianceProfile.Annotations,
			Description:    complianceProfile.Description,
			Title:          complianceProfile.Title,
			OperatorKind:   central.ComplianceOperatorProfileV2_PROFILE,
		}

		for _, r := range complianceProfile.Rules {
			protoProfile.Rules = append(protoProfile.Rules, &central.ComplianceOperatorProfileV2_Rule{
				RuleName: string(r),
			})
		}

		for _, v := range complianceProfile.Values {
			protoProfile.Values = append(protoProfile.Values, string(v))
		}

		events = append(events, &central.SensorEvent{
			Id:     uid,
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorProfileV2{
				ComplianceOperatorProfileV2: protoProfile,
			},
		})
	}

	return component.NewEvent(events...)
}
