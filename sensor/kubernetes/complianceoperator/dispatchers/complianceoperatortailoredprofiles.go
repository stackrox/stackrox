package dispatchers

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

// TailoredProfileDispatcher handles compliance operator tailored profile objects
type TailoredProfileDispatcher struct {
	profileLister cache.GenericLister
}

// NewTailoredProfileDispatcher creates and returns a new tailored profile dispatcher
func NewTailoredProfileDispatcher(profileLister cache.GenericLister) *TailoredProfileDispatcher {
	return &TailoredProfileDispatcher{
		profileLister: profileLister,
	}
}

// ProcessEvent processes a compliance operator tailored profile
func (c *TailoredProfileDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	var tailoredProfile v1alpha1.TailoredProfile

	unstructuredObject, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Not of type 'unstructured': %T", obj)
		return nil
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.Object, &tailoredProfile); err != nil {
		log.Errorf("error converting unstructured to tailored compliance profile: %v", err)
		return nil
	}

	if tailoredProfile.Status.ID == "" {
		log.Warnf("Tailored profile %s does not have an ID. Skipping...", tailoredProfile.Name)
		return nil
	}

	var baseProfile v1alpha1.Profile
	if tailoredProfile.Spec.Extends != "" {
		profileObj, err := c.profileLister.ByNamespace(tailoredProfile.GetNamespace()).Get(tailoredProfile.Spec.Extends)
		if err != nil {
			log.Errorf("error getting profile %s: %v", tailoredProfile.Spec.Extends, err)
			return nil
		}
		unstructuredObject, ok = profileObj.(*unstructured.Unstructured)
		if !ok {
			log.Errorf("Fetched profile not of type 'unstructured': %T", profileObj)
			return nil
		}

		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.Object, &baseProfile); err != nil {
			log.Errorf("error converting unstructured to compliance profile: %v", err)
			return nil
		}
	}

	// The compliance operator sets ComplianceScan.Spec.Profile to the tailored profile's
	// k8s name (not its XCCDF Status.ID) for any CEL-based tailored profile
	// (annotation compliance.openshift.io/scanner-type=CEL). This covers:
	//   - TPs with CustomRules (also have CustomRuleProfileAnnotation=true)
	//   - TPs with CEL-typed rules but no CustomRules (ScannerTypeAnnotation only)
	//   - TPs extending a CEL base profile (inherit ScannerTypeAnnotation from base)
	// See https://github.com/ComplianceAsCode/compliance-operator/pull/19214 for the
	// original CustomRuleProfileAnnotation-based fix and the PR that introduced the
	// broader ScannerTypeAnnotation check in CO's scansettingbinding controller.
	// We must use the same value as ProfileId so that BuildProfileRefID produces
	// matching UUIDs on both the profile and the scan sides.
	var profileID string
	switch scannerType := tailoredProfile.GetAnnotations()[v1alpha1.ScannerTypeAnnotation]; {
	case scannerType == string(v1alpha1.ScannerTypeCEL):
		profileID = tailoredProfile.GetName()
	case scannerType == string(v1alpha1.ScannerTypeOpenSCAP), scannerType == "":
		profileID = tailoredProfile.Status.ID
	default:
		profileID = tailoredProfile.Status.ID
		log.Warnf("Tailored profile %s has unrecognised scanner-type annotation %q: using XCCDF ID %q as ProfileId; "+
			"if compliance coverage shows 0 results, this scanner type may need handling here",
			tailoredProfile.GetName(), scannerType, profileID)
	}

	protoProfile := &storage.ComplianceOperatorProfile{
		Id:          string(tailoredProfile.GetUID()),
		ProfileId:   profileID,
		Name:        tailoredProfile.GetName(),
		Labels:      tailoredProfile.GetLabels(),
		Annotations: tailoredProfile.GetAnnotations(),
		Description: tailoredProfile.Spec.Description,
	}

	removedRules := set.NewStringSet()
	for _, rule := range tailoredProfile.Spec.DisableRules {
		removedRules.Add(rule.Name)
	}

	for _, r := range baseProfile.Rules {
		if removedRules.Contains(string(r)) {
			continue
		}
		protoProfile.Rules = append(protoProfile.Rules, &storage.ComplianceOperatorProfile_Rule{
			Name: string(r),
		})
	}
	for _, rule := range tailoredProfile.Spec.EnableRules {
		protoProfile.Rules = append(protoProfile.Rules, &storage.ComplianceOperatorProfile_Rule{
			Name: rule.Name,
		})
	}

	events := []*central.SensorEvent{
		{
			Id:     protoProfile.GetId(),
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorProfile{
				ComplianceOperatorProfile: protoProfile,
			},
		},
	}

	if centralcaps.Has(centralsensor.ComplianceV2TailoredProfiles) {
		protoProfileV2 := &central.ComplianceOperatorProfileV2{
			Id:           protoProfile.GetId(),
			ProfileId:    protoProfile.GetProfileId(),
			Name:         protoProfile.GetName(),
			Labels:       protoProfile.GetLabels(),
			Annotations:  protoProfile.GetAnnotations(),
			Description:  protoProfile.GetDescription(),
			Title:        tailoredProfile.Spec.Title,
			OperatorKind: central.ComplianceOperatorProfileV2_TAILORED_PROFILE,
		}

		for _, rule := range protoProfile.GetRules() {
			protoProfileV2.Rules = append(protoProfileV2.Rules, &central.ComplianceOperatorProfileV2_Rule{RuleName: rule.GetName()})
		}

		events = append(events, &central.SensorEvent{
			Id:     protoProfileV2.GetId(),
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorProfileV2{
				ComplianceOperatorProfileV2: protoProfileV2,
			},
		})
	}

	return component.NewEvent(events...)
}
