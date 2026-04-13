package dispatchers

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	unstructuredObject, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Not of type 'unstructured': %T", obj)
		return nil
	}

	statusID, _, _ := unstructured.NestedString(unstructuredObject.Object, "status", "id")
	if statusID == "" {
		log.Warnf("Tailored profile %s does not have an ID. Skipping...", unstructuredObject.GetName())
		return nil
	}

	// Read base profile rules if this tailored profile extends another profile.
	var baseProfileRules []string
	extendsProfile, _, _ := unstructured.NestedString(unstructuredObject.Object, "spec", "extends")
	if extendsProfile != "" {
		profileObj, err := c.profileLister.ByNamespace(unstructuredObject.GetNamespace()).Get(extendsProfile)
		if err != nil {
			log.Errorf("error getting profile %s: %v", extendsProfile, err)
			return nil
		}
		baseUnstructured, ok := profileObj.(*unstructured.Unstructured)
		if !ok {
			log.Errorf("Fetched profile not of type 'unstructured': %T", profileObj)
			return nil
		}

		// Profile.Rules is []ProfileRule (string alias), serialized as inline JSON field "rules".
		baseProfileRules, _, _ = unstructured.NestedStringSlice(baseUnstructured.Object, "rules")
	}

	// The compliance operator sets ComplianceScan.Spec.Profile to the tailored profile's
	// k8s name (not its XCCDF Status.ID) when the tailored profile contains custom rules
	// (annotation compliance.openshift.io/tailored-profile-contains-custom-rules=true, see
	// https://github.com/ComplianceAsCode/compliance-operator/blob/197c942793f0f0ef81ca39e4e9082271218b8b42/pkg/controller/scansettingbinding/scansettingbinding_controller.go#L555-L563
	// for details). We must use the same value as ProfileId so that BuildProfileRefID
	// produces matching UUIDs on both the profile and the scan sides.
	profileID := statusID
	if unstructuredObject.GetAnnotations()[customRuleProfileAnnotation] == "true" {
		profileID = unstructuredObject.GetName()
	}

	description, _, _ := unstructured.NestedString(unstructuredObject.Object, "spec", "description")

	protoProfile := &storage.ComplianceOperatorProfile{
		Id:          string(unstructuredObject.GetUID()),
		ProfileId:   profileID,
		Name:        unstructuredObject.GetName(),
		Labels:      unstructuredObject.GetLabels(),
		Annotations: unstructuredObject.GetAnnotations(),
		Description: description,
	}

	removedRules := set.NewStringSet()
	disableRulesList, _, _ := unstructured.NestedSlice(unstructuredObject.Object, "spec", "disableRules")
	for _, rule := range disableRulesList {
		if ruleMap, ok := rule.(map[string]interface{}); ok {
			if name, ok := ruleMap["name"].(string); ok {
				removedRules.Add(name)
			}
		}
	}

	for _, r := range baseProfileRules {
		if removedRules.Contains(r) {
			continue
		}
		protoProfile.Rules = append(protoProfile.Rules, &storage.ComplianceOperatorProfile_Rule{
			Name: r,
		})
	}

	enableRulesList, _, _ := unstructured.NestedSlice(unstructuredObject.Object, "spec", "enableRules")
	for _, rule := range enableRulesList {
		if ruleMap, ok := rule.(map[string]interface{}); ok {
			if name, ok := ruleMap["name"].(string); ok {
				protoProfile.Rules = append(protoProfile.Rules, &storage.ComplianceOperatorProfile_Rule{
					Name: name,
				})
			}
		}
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
		title, _, _ := unstructured.NestedString(unstructuredObject.Object, "spec", "title")

		protoProfileV2 := &central.ComplianceOperatorProfileV2{
			Id:           protoProfile.GetId(),
			ProfileId:    protoProfile.GetProfileId(),
			Name:         protoProfile.GetName(),
			Labels:       protoProfile.GetLabels(),
			Annotations:  protoProfile.GetAnnotations(),
			Description:  protoProfile.GetDescription(),
			Title:        title,
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
